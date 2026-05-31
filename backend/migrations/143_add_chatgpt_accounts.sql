-- 143_add_chatgpt_accounts.sql
-- Add ChatGPT account pool and image generation tables

-- ChatGPT accounts table
CREATE TABLE IF NOT EXISTS chatgpt_accounts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    session_token TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    shared BOOLEAN NOT NULL DEFAULT false,
    plus BOOLEAN NOT NULL DEFAULT false,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chatgpt_accounts_status ON chatgpt_accounts(status);
CREATE INDEX IF NOT EXISTS idx_chatgpt_accounts_email ON chatgpt_accounts(email);
CREATE INDEX IF NOT EXISTS idx_chatgpt_accounts_last_used_at ON chatgpt_accounts(last_used_at);

-- ChatGPT image generation storage table
CREATE TABLE IF NOT EXISTS chatgpt_images (
    id BIGSERIAL PRIMARY KEY,
    prompt TEXT NOT NULL,
    model VARCHAR(50) NOT NULL,
    image_url TEXT NOT NULL,
    revised_prompt TEXT,
    account_id BIGINT REFERENCES chatgpt_accounts(id) ON DELETE SET NULL,
    user_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chatgpt_images_user_id ON chatgpt_images(user_id);
CREATE INDEX IF NOT EXISTS idx_chatgpt_images_account_id ON chatgpt_images(account_id);
CREATE INDEX IF NOT EXISTS idx_chatgpt_images_created_at ON chatgpt_images(created_at);
