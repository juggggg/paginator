# 分页说明：游标分页 + 延迟关联跳页

本模块提供基于 GORM 的高性能分页方案，结合了游标分页（前后翻页）和延迟关联分页（跳页）两种模式，兼顾性能和功能完整性。

---

## ✨ 功能概述

- **游标分页**：通过游标（Token）支持前后翻页，适合上下滑动浏览。
- **延迟关联跳页**：当用户跳转至指定页码时启用，避免深分页带来的性能问题。

---

## 📦 使用示例

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
