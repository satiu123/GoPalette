export interface BlogSection {
  title: string
  paragraphs: string[]
  quote?: string
  bullets?: string[]
}

export interface BlogPost {
  slug: string
  title: string
  summary: string
  cover: string
  category: string
  tags: string[]
  author: string
  publishedAt: string
  readingMinutes: number
  featured?: boolean
  sections: BlogSection[]
}

export const blogPosts: BlogPost[] = [
  {
    slug: 'go-service-observability-playbook',
    title: 'Go 微服务可观测性落地手册',
    summary: '从日志、链路到指标，建立可回溯、可预警、可定位的服务治理闭环。',
    cover: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1600&q=80',
    category: '后端工程',
    tags: ['Go', 'OpenTelemetry', 'SRE'],
    author: 'Satiu',
    publishedAt: '2026-04-10',
    readingMinutes: 8,
    featured: true,
    sections: [
      {
        title: '为什么先做可观测性',
        paragraphs: [
          '多数线上故障不是因为没有修复能力，而是缺少快速定位能力。可观测性让系统内部状态能够被外部推断。',
          '在 Go 服务体系中，日志、指标和 Trace 应作为同一套上下文体系管理，而不是三套孤立数据源。'
        ]
      },
      {
        title: '最小可用方案',
        paragraphs: ['从请求入口注入 trace_id，统一结构化日志输出，并暴露核心 SLI 指标。'],
        bullets: ['日志统一 JSON Schema', '指标聚焦错误率、时延、吞吐', '链路追踪覆盖入口、数据库、消息队列']
      }
    ]
  },
  {
    slug: 'nuxt-editor-for-content-workflow',
    title: '用 Nuxt Editor 构建内容工作流',
    summary: '把编辑器从 demo 提升为生产可用的博客写作中台，统一发布体验。',
    cover: 'https://images.unsplash.com/photo-1461749280684-dccba630e2f6?auto=format&fit=crop&w=1600&q=80',
    category: '前端架构',
    tags: ['Nuxt', 'TipTap', 'UX'],
    author: 'Satiu',
    publishedAt: '2026-04-08',
    readingMinutes: 6,
    featured: true,
    sections: [
      {
        title: '从模板到系统',
        paragraphs: [
          '编辑器模板通常具备强大的输入能力，但缺少发布链路。完善博客前端要补齐列表、详情、导航和内容索引。',
          '保持风格一致的关键是复用同一套 header、间距尺度和组件语义。'
        ],
        quote: '一个可维护的内容平台，不是页面堆砌，而是信息结构一致。'
      },
      {
        title: '实践建议',
        paragraphs: ['优先建立内容数据模型，然后让首页、列表页、详情页围绕同一数据驱动渲染。']
      }
    ]
  },
  {
    slug: 'grpc-http-gateway-design',
    title: 'gRPC + HTTP Gateway 的接口演进策略',
    summary: '在兼顾性能与易用性的同时，保证 API 的版本稳定性与可扩展性。',
    cover: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=1600&q=80',
    category: '接口设计',
    tags: ['gRPC', 'API', 'Gateway'],
    author: 'Satiu',
    publishedAt: '2026-04-05',
    readingMinutes: 9,
    sections: [
      {
        title: '协议分层',
        paragraphs: ['内部服务坚持 gRPC，面向外部保留 HTTP Gateway，可以最大化复用领域能力。']
      },
      {
        title: '版本治理',
        paragraphs: ['确保 proto 兼容演进，使用版本命名空间隔离破坏性变更。'],
        bullets: ['避免字段复用语义', '新增优于修改', '废弃字段保留窗口']
      }
    ]
  },
  {
    slug: 'search-service-indexing-basics',
    title: '搜索服务索引构建基础',
    summary: '从分词、倒排到相关性排序，快速搭建可迭代的搜索能力。',
    cover: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=1600&q=80',
    category: '搜索系统',
    tags: ['Search', 'Index', 'Ranking'],
    author: 'Satiu',
    publishedAt: '2026-04-02',
    readingMinutes: 7,
    sections: [
      {
        title: '索引生命周期',
        paragraphs: ['把索引看作可重建资产：构建、增量更新、回滚与重放策略同样重要。']
      }
    ]
  }
]

export function getAllTags(): string[] {
  return Array.from(new Set(blogPosts.flatMap(post => post.tags))).sort((a, b) => a.localeCompare(b, 'zh-CN'))
}

export function getPostBySlug(slug: string): BlogPost | undefined {
  return blogPosts.find(post => post.slug === slug)
}
