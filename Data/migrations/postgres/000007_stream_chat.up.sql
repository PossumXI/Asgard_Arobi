-- Stream chat messages
CREATE TABLE IF NOT EXISTS stream_chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id UUID,
    username TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stream_chat_stream_id ON stream_chat_messages(stream_id);
CREATE INDEX IF NOT EXISTS idx_stream_chat_created_at ON stream_chat_messages(created_at);
