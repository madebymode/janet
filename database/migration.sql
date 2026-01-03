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
WHERE from_user NOT GLOB '[a-zA-Z0-9._-]*'
   OR from_user = ''
   OR LENGTH(from_user) > 50
   OR to_user NOT GLOB '[a-zA-Z0-9._-]*'
   OR to_user = ''
   OR LENGTH(to_user) > 50;

