CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TABLE users_table
(
  id SERIAL PRIMARY KEY,
  user_name text not null,
  user_email text not null,
  user_oauth_info text not null,
  user_subscriber_status boolean default 0,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

CREATE TRIGGER items_updated_at_timestamp
BEFORE UPDATE ON items_table
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE VIEW users
AS
  SELECT
    id,
    user_name,
    user_email,
    user_oauth_info,
    user_subscriber_status,
    created_at,
    updated_at
  FROM
    users_table;


CREATE TABLE items_table
(
  id SERIAL PRIMARY KEY,
  user_id integer REFERENCES users_table(id) ON DELETE CASCADE,
  plaid_access_token text UNIQUE NOT NULL,
  plaid_item_id text UNIQUE NOT NULL,
  plaid_institution_id text NOT NULL,
  status text NOT NULL,
  created_at timestamptz default now(),
  updated_at timestamptz default now(),
  transactions_cursor text
);

CREATE TRIGGER items_updated_at_timestamp
BEFORE UPDATE ON items_table
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE VIEW items
AS
  SELECT
    id,
    plaid_item_id,
    user_id,
    plaid_access_token,
    plaid_institution_id,
    status,
    created_at,
    updated_at,
    transactions_cursor
  FROM
    items_table;


CREATE TABLE assets_table
(
  id SERIAL PRIMARY KEY,
  user_id integer REFERENCES users_table(id),
  value numeric(28,2),
  description text,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

CREATE TRIGGER assets_updated_at_timestamp
BEFORE UPDATE ON assets_table
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE VIEW assets
AS
  SELECT
    id,
    user_id,
    value,
    description,
    created_at,
    updated_at
  FROM
    assets_table;


CREATE TABLE accounts_table
(
  id SERIAL PRIMARY KEY,
  item_id integer REFERENCES items_table(id) ON DELETE CASCADE,
  plaid_account_id text UNIQUE NOT NULL,
  name text NOT NULL,
  mask text NOT NULL,
  official_name text,
  current_balance numeric(28,10),
  available_balance numeric(28,10),
  iso_currency_code text,
  unofficial_currency_code text,
  type text NOT NULL,
  subtype text NOT NULL,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

CREATE TRIGGER accounts_updated_at_timestamp
BEFORE UPDATE ON accounts_table
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE VIEW accounts
AS
  SELECT
    a.id,
    a.plaid_account_id,
    a.item_id,
    i.plaid_item_id,
    i.user_id,
    a.name,
    a.mask,
    a.official_name,
    a.current_balance,
    a.available_balance,
    a.iso_currency_code,
    a.unofficial_currency_code,
    a.type,
    a.subtype,
    a.created_at,
    a.updated_at
  FROM
    accounts_table a
    LEFT JOIN items i ON i.id = a.item_id;


CREATE TABLE transactions_table
(
  id SERIAL PRIMARY KEY,
  account_id integer REFERENCES accounts_table(id) ON DELETE CASCADE,
  plaid_transaction_id text UNIQUE NOT NULL,
  plaid_category_id text,
  category text,
  subcategory text,
  type text NOT NULL,
  name text NOT NULL,
  amount numeric(28,10) NOT NULL,
  iso_currency_code text,
  unofficial_currency_code text,
  date date NOT NULL,
  pending boolean NOT NULL,
  account_owner text,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

CREATE TRIGGER transactions_updated_at_timestamp
BEFORE UPDATE ON transactions_table
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE VIEW transactions
AS
  SELECT
    t.id,
    t.plaid_transaction_id,
    t.account_id,
    a.plaid_account_id,
    a.item_id,
    a.plaid_item_id,
    a.user_id,
    t.category,
    t.subcategory,
    t.type,
    t.name,
    t.amount,
    t.iso_currency_code,
    t.unofficial_currency_code,
    t.date,
    t.pending,
    t.account_owner,
    t.created_at,
    t.updated_at
  FROM
    transactions_table t
    LEFT JOIN accounts a ON t.account_id = a.id;


CREATE TABLE link_events_table
(
  id SERIAL PRIMARY KEY,
  type text NOT NULL,
  user_id integer,
  link_session_id text,
  request_id text UNIQUE,
  error_type text,
  error_code text,
  status text,
  created_at timestamptz default now()
);


CREATE TABLE plaid_api_events_table
(
  id SERIAL PRIMARY KEY,
  item_id integer,
  user_id integer,
  plaid_method text NOT NULL,
  arguments text,
  request_id text UNIQUE,
  error_type text,
  error_code text,
  created_at timestamptz default now()
);