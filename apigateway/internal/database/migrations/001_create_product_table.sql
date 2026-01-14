-- Product and Feature tables for Product Management System

-- Product table
CREATE TABLE IF NOT EXISTS product (
    id BIGINT NOT NULL,
    brand VARCHAR(255) NOT NULL,
    revision BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (brand, id)
);

CREATE INDEX idx_product_id ON product(id);
CREATE INDEX idx_product_brand ON product(brand);

-- Feature table (stores features for each product-country combination)
CREATE TABLE IF NOT EXISTS feature (
    id BIGINT NOT NULL,
    brand VARCHAR(255) NOT NULL,
    country VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    sub_number INTEGER NOT NULL,
    PRIMARY KEY (brand, id, country, sub_number),
    FOREIGN KEY (brand, id) REFERENCES product(brand, id) ON DELETE CASCADE
);

CREATE INDEX idx_feature_product ON feature(brand, id);
CREATE INDEX idx_feature_country ON feature(country);
CREATE INDEX idx_feature_brand ON feature(brand);
