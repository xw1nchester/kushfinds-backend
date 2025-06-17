CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_user_agent_user_id
ON sessions (user_agent, user_id);