CREATE TABLE IF NOT EXISTS loan_customers (
    customer_id    TEXT    PRIMARY KEY,
    id_card_number TEXT    NOT NULL UNIQUE,
    full_name      TEXT    NOT NULL,
    birth_date     TEXT    NOT NULL,
    phone_number   TEXT    NOT NULL,
    email          TEXT,
    monthly_income INTEGER NOT NULL DEFAULT 0,
    address_street TEXT    NOT NULL,
    address_city   TEXT    NOT NULL
);
