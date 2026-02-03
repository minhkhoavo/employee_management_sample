package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// printMemUsage in ra RAM hiện tại (tính bằng MiB)
// Giúp theo dõi mức sử dụng bộ nhớ tại các điểm khác nhau trong chương trình
func printMemUsage(step string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Alloc: RAM đang thực sự chứa dữ liệu biến
	// Sys: RAM mà OS đã cấp cho chương trình (thường lớn hơn Alloc)
	fmt.Printf("[%s] Alloc = %v MiB \t TotalSys = %v MiB \t NumGC = %v\n",
		step, m.Alloc/1024/1024, m.Sys/1024/1024, m.NumGC)
}

// TraceFunction là wrapper để profiling một hàm cụ thể
// Nó ghi lại memory profile trước và sau khi hàm chạy
// Cho phép phân tích chi tiết RAM consumption
func TraceFunction(name string, fn func()) {
	fmt.Printf("\n========== TRACING: %s ==========\n", name)

	// 1. Chụp RAM lúc bắt đầu
	printMemUsage("START " + name)

	// 2. Tạo file heap profile trước khi chạy
	f1, err := os.Create(name + "_mem_before.pprof")
	if err != nil {
		fmt.Printf("Error creating before profile: %v\n", err)
		return
	}
	pprof.WriteHeapProfile(f1)
	f1.Close()
	fmt.Printf("[%s] Heap profile (before) saved to: %s_mem_before.pprof\n", name, name)

	// 3. Chạy hàm nặng
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	fmt.Printf("[%s] Execution Time: %s\n", name, elapsed)

	// 4. Ép dọn rác để xem RAM thực tế còn lại
	runtime.GC()
	printMemUsage("AFTER GC " + name)

	// 5. Chụp RAM lần nữa sau khi chạy xong
	f2, err := os.Create(name + "_mem_after.pprof")
	if err != nil {
		fmt.Printf("Error creating after profile: %v\n", err)
		return
	}
	pprof.WriteHeapProfile(f2)
	f2.Close()
	fmt.Printf("[%s] Heap profile (after) saved to: %s_mem_after.pprof\n", name, name)

	fmt.Printf("========== END TRACING: %s ==========\n\n", name)
}

// processHeavyDataTask là hàm giả lập xử lý data nặng (như đọc Excel lớn)
// Nó sẽ cấp phát một lượng lớn bộ nhớ để mô phỏng tình huống thực tế
func processHeavyDataTask() {
	// Cấp phát một danh sách lớn để mô phỏng việc đọc file Excel
	// Mỗi phần tử là một chuỗi dữ liệu (giả lập một cell)
	const rowCount = 100000
	const colCount = 50

	// Giai đoạn 1: Tải data vào memory
	fmt.Println("[processHeavyDataTask] Loading data into memory...")
	data := make([][]string, rowCount)
	for i := 0; i < rowCount; i++ {
		data[i] = make([]string, colCount)
		for j := 0; j < colCount; j++ {
			// Mỗi cell chứa 100 bytes dữ liệu
			data[i][j] = fmt.Sprintf("Cell_%d_%d_with_some_data_to_simulate_real_world_%d", i, j, i*j)
		}
	}
	fmt.Printf("[processHeavyDataTask] Loaded %d rows x %d columns\n", rowCount, colCount)

	// Giai đoạn 2: Xử lý data (giả lập)
	fmt.Println("[processHeavyDataTask] Processing data...")
	totalLength := 0
	for i := 0; i < rowCount; i++ {
		for j := 0; j < colCount; j++ {
			totalLength += len(data[i][j])
		}
	}
	fmt.Printf("[processHeavyDataTask] Processed data. Total length: %d\n", totalLength)

	// Giai đoạn 3: Cấp phát thêm memory cho tính toán
	fmt.Println("[processHeavyDataTask] Allocating additional memory...")
	tempBuffer := make([][]int64, rowCount)
	for i := 0; i < rowCount; i++ {
		tempBuffer[i] = make([]int64, colCount)
		for j := 0; j < colCount; j++ {
			tempBuffer[i][j] = int64(i*colCount + j)
		}
	}
	fmt.Println("[processHeavyDataTask] Processing completed!")

	// Lưu ý: data và tempBuffer sẽ bị giải phóng tự động khi hàm kết thúc
	// Nhưng garbage collector chưa chạy, nên heap profile sẽ capture chúng
}

// calculateCombinations là hàm thứ hai để test, tính toán tổ hợp
func calculateCombinations() {
	fmt.Println("[calculateCombinations] Starting calculations...")

	const size = 50000
	numbers := make([]int, size)
	for i := 0; i < size; i++ {
		numbers[i] = i * i
	}

	// Tính toán một số thống kê
	sum := int64(0)
	for _, n := range numbers {
		sum += int64(n)
	}
	fmt.Printf("[calculateCombinations] Sum: %d\n", sum)

	// Cấp phát thêm
	results := make(map[int]int64)
	for i, n := range numbers {
		results[i] = int64(n * n)
	}
	fmt.Printf("[calculateCombinations] Map entries: %d\n", len(results))
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	fmt.Println("Go Memory Profiling Demo with pprof CLI")
	fmt.Println(strings.Repeat("=", 50))

	// Test hàm nặng thứ nhất
	TraceFunction("HeavyDataTask", processHeavyDataTask)

	// Chờ một chút để heap profile hoàn toàn ghi xuống disk
	time.Sleep(100 * time.Millisecond)

	// Test hàm thứ hai
	TraceFunction("CombinationCalc", calculateCombinations)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Profiling completed!")
	fmt.Println("\nNext steps - Analyze profiles with these commands in terminal:\n")
	fmt.Println("1. View top 10 memory consumers (before execution):")
	fmt.Println("   go tool pprof -top -cum HeavyDataTask_mem_before.pprof")
	fmt.Println("\n2. View top 10 memory consumers (after execution):")
	fmt.Println("   go tool pprof -top -cum HeavyDataTask_mem_after.pprof")
	fmt.Println("\n3. Interactive analysis (type 'list processHeavyDataTask' inside pprof):")
	fmt.Println("   go tool pprof HeavyDataTask_mem_after.pprof")
	fmt.Println("\n4. Compare two profiles to see difference:")
	fmt.Println("   go tool pprof -base=HeavyDataTask_mem_before.pprof HeavyDataTask_mem_after.pprof")
}

// func DownloadExcel(c echo.Context) error {
//     ctx := c.Request().Context()

//     // 1️⃣ Lấy data (ví dụ)
//     rows, err := repo.GetBigReport(ctx) // []YourStruct
//     if err != nil {
//         return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
//     }

//     // 2️⃣ Tạo file Excel
//     f := excelize.NewFile()
//     sheet := "Report"
//     f.SetSheetName("Sheet1", sheet)

//     // 3️⃣ Header
//     headers := []string{"ID", "Name", "Amount", "CreatedAt"}
//     for i, h := range headers {
//         cell, _ := excelize.CoordinatesToCellName(i+1, 1)
//         f.SetCellValue(sheet, cell, h)
//     }

//     // 4️⃣ Ghi data (⚠️ KHÔNG gom vào buffer)
//     for i, r := range rows {
//         rowIdx := i + 2

//         f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.ID)
//         f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Name)
//         f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Amount)
//         f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.CreatedAt.Format(time.RFC3339))
//     }

//     // ============================
//     // 🔥 OPTIONAL: SNAPSHOT HEAP
//     // ============================
//     if c.QueryParam("pprof") == "1" {
//         runtime.GC()
//         fheap, _ := os.Create("/tmp/heap-excel.out")
//         pprof.WriteHeapProfile(fheap)
//         fheap.Close()
//     }

//     // 5️⃣ Set header response
//     res := c.Response()
//     res.Header().Set(echo.HeaderContentType,
//         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
//     res.Header().Set(echo.HeaderContentDisposition,
//         `attachment; filename="report.xlsx"`)

//     // 6️⃣ STREAM TRỰC TIẾP RA CLIENT 🚀
//     res.WriteHeader(http.StatusOK)

//     if err := f.Write(res.Writer); err != nil {
//         return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
//     }

//     return nil
// }
