ALTER TABLE relationship_user_relationship_counters
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_friends_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_followers_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_following_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_blocked_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_pending_in_nonnegative,
    DROP CONSTRAINT IF EXISTS chk_relationship_counters_pending_out_nonnegative;

DROP TABLE IF EXISTS relationship_pair_guards;
