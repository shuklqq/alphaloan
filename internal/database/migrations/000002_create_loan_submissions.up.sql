CREATE TABLE IF NOT EXISTS loan_submissions (
    submission_id              TEXT    PRIMARY KEY,
    vehicle_type               TEXT    NOT NULL,
    vehicle_brand              TEXT    NOT NULL,
    vehicle_model              TEXT    NOT NULL,
    vehicle_license_number     TEXT    NOT NULL,
    vehicle_odometer           INTEGER NOT NULL DEFAULT 0,
    manufacturing_year         INTEGER NOT NULL,
    proposed_loan_amount       INTEGER NOT NULL DEFAULT 0,
    proposed_loan_tenure_month INTEGER NOT NULL DEFAULT 0,
    loan_status                TEXT    NOT NULL,
    is_commercial_vehicle      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at                 INTEGER NOT NULL,
    updated_at                 INTEGER NOT NULL,
    customer_id                TEXT    NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES loan_customers (customer_id) ON DELETE CASCADE
);
