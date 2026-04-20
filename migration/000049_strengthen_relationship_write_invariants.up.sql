CREATE TABLE relationship_pair_guards (
    user_low_id     VARCHAR(36) NOT NULL,
    user_high_id    VARCHAR(36) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT pk_relationship_pair_guards
        PRIMARY KEY (user_low_id, user_high_id),
    CONSTRAINT chk_relationship_pair_guards_order
        CHECK (user_low_id < user_high_id)
);

ALTER TABLE relationship_user_relationship_counters
    ADD CONSTRAINT chk_relationship_counters_friends_nonnegative CHECK (friends_count >= 0),
    ADD CONSTRAINT chk_relationship_counters_followers_nonnegative CHECK (followers_count >= 0),
    ADD CONSTRAINT chk_relationship_counters_following_nonnegative CHECK (following_count >= 0),
    ADD CONSTRAINT chk_relationship_counters_blocked_nonnegative CHECK (blocked_count >= 0),
    ADD CONSTRAINT chk_relationship_counters_pending_in_nonnegative CHECK (pending_in_count >= 0),
    ADD CONSTRAINT chk_relationship_counters_pending_out_nonnegative CHECK (pending_out_count >= 0);
