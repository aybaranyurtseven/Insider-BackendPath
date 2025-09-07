CREATE TABLE IF NOT EXISTS balances (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(15,2) NOT NULL DEFAULT 0.00 CHECK (amount >= 0),
    last_updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_balances_last_updated_at ON balances(last_updated_at);
CREATE INDEX idx_balances_amount ON balances(amount);
