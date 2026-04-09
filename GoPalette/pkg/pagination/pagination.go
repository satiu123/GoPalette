package pagination

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
	DefaultPageNum  = 1
)

// PagingParam 分页请求参数
type PagingParam struct {
	PageNum  int64
	PageSize int64
}

// GetOffset 计算数据库 offset
func (p *PagingParam) GetOffset() int {
	if p.PageNum <= 0 {
		p.PageNum = DefaultPageNum
	}
	return int((p.PageNum - 1) * p.GetLimit())
}

// GetLimit 获取每页限制量，防止被恶意抓取大量数据
func (p *PagingParam) GetLimit() int64 {
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	return p.PageSize
}

// NewPagingParam 构造函数，可处理 proto 生成的各种类型
func NewPagingParam(page, pageSize int64) *PagingParam {
	return &PagingParam{
		PageNum:  page,
		PageSize: pageSize,
	}
}

// PageResult 统一返回格式
type PageResult[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}

func NewPageResult[T any](total int64, list []T) *PageResult[T] {
	return &PageResult[T]{
		Total: total,
		List:  list,
	}
}
