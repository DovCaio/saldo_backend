ALTER TABLE transactions
ADD COLUMN category VARCHAR(50),
ADD COLUMN category_status VARCHAR(20) NOT NULL DEFAULT 'pending'
CHECK (category_status IN ('pending', 'completed', 'failed'));