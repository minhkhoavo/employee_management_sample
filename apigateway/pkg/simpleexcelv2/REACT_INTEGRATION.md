# React v17 Integration Guide

This guide demonstrates how to download large Excel files from the `simpleexcel` Go backend using React v17. It focuses on memory efficiency and handling potential timeouts.

## Core Concepts

1.  **Blob Handling**: Large files should be handled as Blobs to prevent browser memory issues.
2.  **ObjectURL Cleanup**: Always revoke created object URLs to prevent memory leaks.
3.  **Timeouts**: Use `AbortController` (Fetch API) or specialized timeout settings (Axios).
4.  **Download Tracking**: For very large files, provide visual feedback (loading state).

## Implementation with Axios (Recommended)

Axios is recommended for large downloads because of its built-in `onDownloadProgress` and easy timeout configuration.

```javascript
import React, { useState } from 'react';
import axios from 'axios';

const ExcelDownloadButton = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleDownload = async () => {
    setLoading(true);
    setError(null);

    // Create an AbortController for manual cancellation or timeout
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minute timeout

    try {
      const response = await axios({
        url: 'http://localhost:8080/api/v2/employees/export/large', // Your endpoint
        method: 'GET',
        responseType: 'blob', // IMPORTANT: Handle response as a Blob
        signal: controller.signal,
        timeout: 300000, // External timeout (5 minutes)
        onDownloadProgress: (progressEvent) => {
          const percentage = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          console.log(`Download progress: ${percentage}%`);
        }
      });

      clearTimeout(timeoutId);

      // Create a link element to trigger the download
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      
      // Extract filename from Content-Disposition if available, or use default
      const contentDisposition = response.headers['content-disposition'];
      let filename = 'export.xlsx';
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="(.+)"/);
        if (filenameMatch && filenameMatch.length > 1) {
          filename = filenameMatch[1];
        }
      }
      
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();

      // CLEANUP: Important for memory management
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      
    } catch (err) {
      if (err.name === 'AbortError' || axios.isCancel(err)) {
        setError('Download timed out or was cancelled.');
      } else {
        setError('Failed to download Excel file. Please try again.');
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <button 
        onClick={handleDownload} 
        disabled={loading}
        className="download-btn"
      >
        {loading ? 'Generating Report...' : 'Download Large Excel'}
      </button>
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
};

export default ExcelDownloadButton;
```

## Implementation with Fetch API

If you prefer not to use Axios, you can use the native `Fetch` API with `AbortController`.

```javascript
const downloadWithFetch = async () => {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minutes

  try {
    const response = await fetch('/api/v2/employees/export/large', {
      method: 'GET',
      signal: controller.signal,
    });

    if (!response.ok) throw new Error('Network response was not ok');

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    
    const a = document.createElement('a');
    a.href = url;
    a.download = 'export.xlsx';
    document.body.appendChild(a);
    a.click();
    
    // Memory safety: cleanup
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  } catch (err) {
    console.error('Fetch download failed', err);
  } finally {
    clearTimeout(timeoutId);
  }
};
```

## Performance & Memory Checklist

- [ ] **Response Type**: Always set `responseType: 'blob'` (Axios) or use `response.blob()` (Fetch).
- [ ] **Revoke ObjectURL**: `window.URL.revokeObjectURL(url)` is critical. Without it, the browser keeps the file data in memory until the tab is closed.
- [ ] **Timeout**: Large exports can take minutes. Ensure your frontend timeout is longer than your backend processing time.
- [ ] **Chunking**: For truly massive data (millions of rows), consider using the **CSV export** mode (`ToCSV`) to reduce the browser's parsing overhead.
- [ ] **Background Processing**: If a download takes more than 1 minute, consider moving to an "Export Job" pattern where the backend generates the file, stores it, and notifies the user via WebSocket or polling.
