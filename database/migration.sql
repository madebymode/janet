-- New unified table database schema for janet (PostgreSQL version)
-- This schema uses a single table for all karma transactions, with a 'year' column for partitioning/filtering.

-- Unified karma transactions table
CREATE TABLE IF NOT EXISTS karma_transactions
(
  id
  SERIAL
  PRIMARY
  KEY,
  from_user
  TEXT
  NOT
  NULL,
  to_user
  TEXT
  NOT
  NULL,
  points
  INTEGER
  NOT
  NULL,
  reason
  TEXT,
  transaction_type
  TEXT
  NOT
  NULL
  DEFAULT
  'manual',
  emoji_name
  TEXT,
  channel_id
  TEXT,
  channel_name
  TEXT,
  message_id
  TEXT,
  timestamp
  TIMESTAMPTZ
  NOT
  NULL
  DEFAULT
  NOW
(
),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW
(
),
  year INTEGER NOT NULL -- New column to store the year of the transaction
  );



-- Indexes for performance
-- Note: Yearly table indexes will be created dynamically for each year

CREATE INDEX IF NOT EXISTS idx_karma_transactions_to_user ON karma_transactions(to_user);
CREATE INDEX IF NOT EXISTS idx_karma_transactions_from_user ON karma_transactions(from_user);
CREATE INDEX IF NOT EXISTS idx_karma_transactions_timestamp ON karma_transactions(timestamp);
CREATE INDEX IF NOT EXISTS idx_karma_transactions_emoji ON karma_transactions(emoji_name) WHERE emoji_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_karma_transactions_type ON karma_transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_karma_transactions_year ON karma_transactions(year);

-- Cached popular message details to avoid repeated Slack API calls
CREATE TABLE IF NOT EXISTS popular_message_cache
(
  message_id TEXT PRIMARY KEY,
  channel_id TEXT,
  message_text TEXT,
  permalink TEXT,
  author_id TEXT,
  author_name TEXT,
  author_avatar TEXT,
  image_url TEXT,
  attachment_url TEXT,
  attachment_mime TEXT,
  slack_reaction_count INTEGER,
  is_reply BOOLEAN DEFAULT FALSE,
  is_ignored BOOLEAN DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE IF EXISTS popular_message_cache
  ADD COLUMN IF NOT EXISTS channel_id TEXT,
  ADD COLUMN IF NOT EXISTS message_text TEXT,
  ADD COLUMN IF NOT EXISTS permalink TEXT,
  ADD COLUMN IF NOT EXISTS author_id TEXT,
  ADD COLUMN IF NOT EXISTS author_name TEXT,
  ADD COLUMN IF NOT EXISTS author_avatar TEXT,
  ADD COLUMN IF NOT EXISTS image_url TEXT,
  ADD COLUMN IF NOT EXISTS attachment_url TEXT,
  ADD COLUMN IF NOT EXISTS attachment_mime TEXT,
  ADD COLUMN IF NOT EXISTS slack_reaction_count INTEGER,
  ADD COLUMN IF NOT EXISTS is_reply BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_ignored BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_popular_message_cache_updated_at ON popular_message_cache(updated_at);

-- User summary tables
CREATE TABLE IF NOT EXISTS user_summary_current
(
  username
  TEXT
  PRIMARY
  KEY,
  total_points
  INTEGER,
  points_given
  INTEGER,
  points_received
  INTEGER,
  transactions_given
  INTEGER,
  transactions_received
  INTEGER,
  emoji_reactions_given
  INTEGER,
  emoji_reactions_received
  INTEGER,
  last_activity
  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_summary_yearly
(
  username
  TEXT,
  year
  INTEGER,
  total_points
  INTEGER,
  points_given
  INTEGER,
  points_received
  INTEGER,
  transactions_given
  INTEGER,
  transactions_received
  INTEGER,
  emoji_reactions_given
  INTEGER,
  emoji_reactions_received
  INTEGER,
  PRIMARY
  KEY
(
  username,
  year
)
  );

CREATE TABLE IF NOT EXISTS user_summary_monthly
(
  username
  TEXT,
  year
  INTEGER,
  month
  INTEGER,
  total_points
  INTEGER,
  points_given
  INTEGER,
  points_received
  INTEGER,
  transactions_given
  INTEGER,
  transactions_received
  INTEGER,
  emoji_reactions_given
  INTEGER,
  emoji_reactions_received
  INTEGER,
  PRIMARY
  KEY
(
  username,
  year,
  month
)
  );

-- Clean up invalid usernames (Slack special syntax like <!subteam^>, <@U...>, :emoji: patterns, etc.)
-- Only keep usernames with alphanumeric characters, dots, underscores, and hyphens
DELETE FROM karma_transactions
WHERE from_user !~ '^[a-zA-Z0-9._-]+$'
   OR from_user = ''
   OR LENGTH(from_user) > 50
   OR to_user !~ '^[a-zA-Z0-9._-]+$'
   OR to_user = ''
   OR LENGTH(to_user) > 50;

DELETE FROM user_summary_current
WHERE username !~ '^[a-zA-Z0-9._-]+$'
   OR username = ''
   OR LENGTH(username) > 50;

DELETE FROM user_summary_yearly
WHERE username !~ '^[a-zA-Z0-9._-]+$'
   OR username = ''
   OR LENGTH(username) > 50;

DELETE FROM user_summary_monthly
WHERE username !~ '^[a-zA-Z0-9._-]+$'
   OR username = ''
   OR LENGTH(username) > 50;
