CREATE TABLE IF NOT EXISTS notifications_by_id (
    id text PRIMARY KEY,
    account_id text,
    kind text,
    type text,
    group_key text,
    subject text,
    body text,
    is_read boolean,
    read_at timestamp,
    created_at timestamp,
    updated_at timestamp,
    sort_at timestamp,
    room_id text,
    room_name text,
    sender_id text,
    sender_name text,
    message_count int,
    last_message_id text,
    last_message_preview text,
    last_message_at timestamp
);

CREATE TABLE IF NOT EXISTS notifications_by_account (
    account_id text,
    sort_at timestamp,
    id text,
    kind text,
    type text,
    group_key text,
    subject text,
    body text,
    is_read boolean,
    read_at timestamp,
    created_at timestamp,
    updated_at timestamp,
    room_id text,
    room_name text,
    sender_id text,
    sender_name text,
    message_count int,
    last_message_id text,
    last_message_preview text,
    last_message_at timestamp,
    PRIMARY KEY ((account_id), sort_at, id)
) WITH CLUSTERING ORDER BY (sort_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS notification_unread_by_account (
    account_id text,
    notification_id text,
    PRIMARY KEY ((account_id), notification_id)
);

CREATE TABLE IF NOT EXISTS message_notification_groups_by_account (
    account_id text,
    group_key text,
    notification_id text,
    is_read boolean,
    read_at timestamp,
    created_at timestamp,
    updated_at timestamp,
    sort_at timestamp,
    room_id text,
    room_name text,
    sender_id text,
    sender_name text,
    message_count int,
    last_message_id text,
    last_message_preview text,
    last_message_at timestamp,
    subject text,
    body text,
    PRIMARY KEY ((account_id), group_key)
);

CREATE TABLE IF NOT EXISTS push_subscriptions_by_account (
    account_id text,
    endpoint text,
    id text,
    keys text,
    created_at timestamp,
    updated_at timestamp,
    PRIMARY KEY ((account_id), endpoint)
);
