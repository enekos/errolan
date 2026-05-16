# Deferred / Future Optimization Ideas

## Tried and discarded
- **Subqueries instead of IN lists**: SQLite does a terrible job with subqueries for moderate IN sizes. LargeWithViewer regressed +950%. Explicit IN lists with covering indexes are much faster.
- **JOIN for all metadata (votes+avatars+reactions)**: LargeWithViewer regressed +44%. JOIN overhead dominates for moderate sizes. DeepReplies improved because fewer large IN lists.
- **JOIN for reaction_counts + viewer reactions**: Similar pattern - hurts moderate cases, helps very large IN lists.
- **JSON functions (json_group_object/json_group_array)**: Slower than manual GROUP_CONCAT parsing for moderate sizes. SQLite JSON building + Go json.Unmarshal is heavier than strings.Split + strconv.Atoi.

## Promising but not pursued
- **Hybrid metadata query**: Use separate queries for IN < 500 items, JOIN/CTE for IN > 1000 items. DeepReplies is the only case that would benefit.
- **Preallocate Replies slices**: Count replies per parent before stitching. Saves ~150 allocations but negligible time improvement.
- **Batch benchmark setup inserts**: Could reduce setup from ~2s to ~0.5s per benchmark, making iteration faster. Doesn't affect measured behavior.
- **Thread cache**: ThreadBySlug/ThreadByID are called on every request. Cache with invalidation on SetThreadLocked, CreateComment, SetCommentStatus.
- **Denormalize avatar_url into comments**: Eliminates attachAvatars query entirely. Schema change with write overhead.
- **Single recursive CTE for comment tree**: Could reduce top-level + replies to one query, but recursive CTEs in SQLite are tricky with LIMIT/ORDER BY.
