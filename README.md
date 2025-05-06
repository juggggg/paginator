# Pagination: Cursor-based and Deferred Join for Jump-to-Page
# 分页说明：游标分页 + 延迟关联跳页

combining **cursor-based pagination** (for previous/next navigation) with **deferred join pagination** (for jumping to a specific page). It balances speed and flexibility.
结合了游标分页（前后翻页）和延迟关联分页（跳页）两种模式，兼顾性能和功能完整性。

---

## ✨ Features

- **Cursor-based pagination**  
  Ideal for previous/next navigation. Fast and scalable with large datasets.

- **Deferred join pagination**  
  Triggered when jumping to a specific page. Uses key-only queries + join to avoid deep offset performance issues.

- **游标分页**：通过游标（Token）支持前后翻页，适合上下滑动浏览。
- **延迟关联跳页**：当用户跳转至指定页码时启用，避免深分页带来的性能问题。

---

## 📦 Example

```go
pager := page.NewPaginator[model.ActivityCommunityCode](db, int(in.Start), int(in.Limit))
pager.SetPrimaryKeys("tenant_id,code").SetFields("code").SetSequence(false)

data, err := pager.Paginate(in.PrevToken, in.NextToken)
if err != nil {
	return nil, errorxplus.DefaultGormError(l.Logger, err, in)
}

res := &proto.ListCommunityCodesResp{
	Total:     uint64(data.Total),
	List:      make([]*proto.CommunityCode, 0, len(data.Data)),
	NextToken: data.NextToken,
	PrevToken: data.PrevToken,
}
