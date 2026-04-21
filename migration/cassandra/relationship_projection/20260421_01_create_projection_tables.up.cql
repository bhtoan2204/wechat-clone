CREATE TABLE IF NOT EXISTS relationship_pair_projections_by_pair (
    pair_id text PRIMARY KEY,
    user_low_id text,
    user_high_id text,
    pending_request_id text,
    pending_requester_id text,
    pending_addressee_id text,
    pending_request_created_at timestamp,
    friendship_id text,
    friendship_created_at timestamp,
    low_follows_high boolean,
    low_follows_high_at timestamp,
    high_follows_low boolean,
    high_follows_low_at timestamp,
    low_blocks_high boolean,
    low_blocks_high_at timestamp,
    high_blocks_low boolean,
    high_blocks_low_at timestamp,
    created_at timestamp,
    updated_at timestamp
);

CREATE TABLE IF NOT EXISTS relationship_friends_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    PRIMARY KEY ((user_id), counterpart_id)
);

CREATE TABLE IF NOT EXISTS relationship_followers_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    PRIMARY KEY ((user_id), counterpart_id)
);

CREATE TABLE IF NOT EXISTS relationship_following_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    PRIMARY KEY ((user_id), counterpart_id)
);

CREATE TABLE IF NOT EXISTS relationship_blocks_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    PRIMARY KEY ((user_id), counterpart_id)
);

CREATE TABLE IF NOT EXISTS relationship_incoming_requests_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    request_id text,
    PRIMARY KEY ((user_id), counterpart_id)
);

CREATE TABLE IF NOT EXISTS relationship_outgoing_requests_by_user (
    user_id text,
    counterpart_id text,
    created_at timestamp,
    request_id text,
    PRIMARY KEY ((user_id), counterpart_id)
);
