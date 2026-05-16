# Deferred / Future Optimization Ideas

## Tried and discarded
- **Subqueries instead of IN lists**: SQLite does a terrible job with subqueries for moderate IN sizes. LargeWithViewer regressed +950%. Explicit IN lists with covering indexes are much faster.
- **JOIN for all metadata (votes+avatars+reactions)**: LargeWithViewer regressed +44%. JOIN overhead dominates for moderate sizes. DeepReplies improved because fewer large IN lists.
- **JOIN for reaction_counts + viewer reactions**: Similar pattern - hurts moderate cases, helps very large IN lists.
- **JSON functions (json_group_object/json_group_array)**: Slower than manual GROUP_CONCAT parsing for moderate sizes. SQLite JSON building + Go json.Unmarshal is heavier than strings.Split + strconv.Atoi.
- **Aggressive map preallocation (len(top)*50)**: Overallocates map memory (~64KB per call), causing 5-10% regressions for normal cases.
- **Preallocate Replies slices**: Count replies per parent before stitching. Saves ~150 allocations but negligible time improvement.

## Implemented
- Covering indexes for top-level queries (thread_id, parent_id, pinned, score, created_at)
- Covering indexes for metadata (votes, reaction_counts)
- GROUP_CONCAT for reactions to reduce row count
- Zero-allocation custom parsers for GROUP_CONCAT results
- LEFT JOIN votes into comment queries (eliminates attachViewerVotes round-trip)
- Correlated subquery for viewer reactions (combines reaction_counts + reactions into 1 query)
- Cache ListEmojis and reuse for CodesForSite
- Cache gravatarURL results
- Cache inPlaceholders strings
- Cache UserByID
- Additional indexes for non-benchmarked queries (comments.user_id, comments.status+created_at)
- Benchmark setup wrapped in transactions + batched inserts
- CommentByID uses single query with LEFT JOIN votes

## Promising but not pursued
- **Hybrid metadata query**: Use separate queries for IN < 500 items, JOIN/CTE for IN > 1000 items. DeepReplies is the only case that would benefit.
- **Thread cache**: ThreadBySlug/ThreadByID are called on every request. Cache with invalidation on SetThreadLocked, CreateComment, SetCommentStatus.
- **Denormalize avatar_url into comments**: Eliminates attachAvatars query entirely. Schema change with write overhead.
- **Single recursive CTE for comment tree**: Could reduce top-level + replies to one query, but recursive CTEs in SQLite are tricky with LIMIT/ORDER BY.
- **Batch reaction count updates in ToggleReaction**: Currently does UPSERT per reaction. Could batch if multiple reactions toggled at once.
- **Combine top-level + replies query**: Difficult due to different sort orders. Subqueries are bad for SQLite.
- **Add site cache to Store**: siteByKey already cached in API layer.
