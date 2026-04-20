package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultPostCount    = 500_000
	defaultWorkerCount  = 12
	defaultKeywordCount = 3_000
	defaultBatchSize    = 1_000
	minKeywordCount     = 1_000
	maxKeywordCount     = 5_000
)

var (
	chineseTerms = []string{
		"微服务", "数据库", "缓存穿透", "高并发", "容器编排", "可观测性", "消息队列", "分布式事务", "服务网格", "负载均衡",
		"限流", "熔断", "降级", "链路追踪", "日志聚合", "接口幂等", "读写分离", "索引优化", "向量检索", "搜索召回",
		"全文索引", "倒排索引", "分词器", "冷热分层", "弹性扩容", "系统稳定性", "性能压测", "容量评估", "蓝绿发布", "灰度发布",
		"故障注入", "容灾演练", "数据一致性", "SQL优化", "并行计算", "实时计算", "离线计算", "对象存储", "边缘计算", "事件驱动",
		"事务消息", "网络抖动", "延迟抖动", "内存碎片", "GC调优", "线程池", "协程调度", "零拷贝", "批处理", "热点隔离",
	}
	englishTerms = []string{
		"golang", "docker", "kubernetes", "meilisearch", "redis", "mariadb", "grpc", "http", "api", "graphql",
		"gateway", "index", "tokenization", "benchmark", "latency", "throughput", "iops", "cache", "query", "ranking",
		"nlp", "observability", "otel", "prometheus", "grafana", "retry", "backoff", "sharding", "replication", "snapshot",
		"migration", "rollback", "schema", "worker", "pipeline", "queue", "consumer", "producer", "batch", "autoscaling",
	}
	tagPool = []string{
		"后端", "架构", "数据库", "搜索", "性能", "压测", "云原生", "Go", "容器", "分布式",
		"索引", "可观测", "缓存", "网络", "算法", "工程实践", "稳定性", "中间件", "数据平台", "微服务",
	}
	specialFragments = []string{
		"#hot-path", "@p99", "%cache_miss", "&retry", "*priority*", "[trace_id]", "{region=cn}", "(failover)", "|canary|", "~burst~",
	}
	defaultCategories = []string{
		"微服务实践", "数据库优化", "搜索引擎", "分布式系统", "云原生架构",
		"性能压测", "工程效能", "系统稳定性", "可观测性", "后端开发",
	}
)

type Post struct {
	Title   string   `json:"title"`
	Slug    string   `json:"slug"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type generatedPost struct {
	Post     Post
	Keywords []string
}

type keywordFreq struct {
	Keyword string
	Count   int
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "generate":
			if err := runGenerate(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "import":
			if err := runImportAndRebuild(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "-h", "--help", "help":
			printUsage()
			return
		}
	}

	// 兼容旧用法：不带子命令时默认执行 generate。
	if err := runGenerate(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("用法:")
	fmt.Println("  posts.go generate [flags]   生成文章与关键词词库（默认）")
	fmt.Println("  posts.go import [flags]     导入 MariaDB 并触发搜索索引重建")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  go run ./pkg/script/posts.go generate -count 500000")
	fmt.Println("  go run ./pkg/script/posts.go import -dsn 'root:GP123@tcp(127.0.0.1:3306)/gopalette?parseTime=True&loc=Local'")
}

func runGenerate(args []string) error {
	var (
		postCount    int
		workerCount  int
		seed         int64
		outputPath   string
		keywordsPath string
		keywordsNum  int
	)

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.IntVar(&postCount, "count", defaultPostCount, "文章数量")
	fs.IntVar(&workerCount, "workers", defaultWorkerCount, "并发 worker 数")
	fs.Int64Var(&seed, "seed", time.Now().UnixNano(), "随机种子")
	fs.StringVar(&outputPath, "output", "posts.jsonl", "文章输出路径（JSONL）")
	fs.StringVar(&keywordsPath, "keywords-output", "keywords.csv", "搜索词库输出路径（CSV）")
	fs.IntVar(&keywordsNum, "keywords-count", defaultKeywordCount, "搜索词数量（建议 1000-5000）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if postCount <= 0 {
		return fmt.Errorf("count 必须 > 0，当前: %d", postCount)
	}
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if keywordsNum < minKeywordCount {
		keywordsNum = minKeywordCount
	}
	if keywordsNum > maxKeywordCount {
		keywordsNum = maxKeywordCount
	}

	if err := generateDataset(postCount, workerCount, seed, outputPath, keywordsPath, keywordsNum); err != nil {
		return err
	}
	return nil
}

func generateDataset(postCount, workerCount int, seed int64, outputPath, keywordsPath string, keywordsNum int) error {
	postFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文章文件失败: %w", err)
	}
	defer postFile.Close()

	jobs := make(chan int, workerCount*2)
	results := make(chan generatedPost, workerCount*4)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		workerSeed := seed + int64((i+1)*97)
		go func(s int64) {
			defer wg.Done()
			faker := gofakeit.New(s)
			for idx := range jobs {
				results <- buildPost(faker, idx)
			}
		}(workerSeed)
	}

	go func() {
		for i := 1; i <= postCount; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	encoder := json.NewEncoder(postFile)
	keywordCounter := make(map[string]int, keywordsNum*2)
	var processed int64

	for result := range results {
		if err := encoder.Encode(result.Post); err != nil {
			return fmt.Errorf("写入文章失败: %w", err)
		}
		for _, kw := range result.Keywords {
			keywordCounter[kw]++
		}
		n := atomic.AddInt64(&processed, 1)
		if n%10_000 == 0 {
			log.Printf("已生成 %d/%d 篇文章", n, postCount)
		}
	}

	if err := writeKeywordsCSV(keywordsPath, keywordCounter, keywordsNum, seed+1_000_003); err != nil {
		return err
	}

	log.Printf("生成完成: posts=%s, keywords=%s", outputPath, keywordsPath)
	return nil
}

func buildPost(faker *gofakeit.Faker, idx int) generatedPost {
	title, titleTerms := buildTitle(faker)
	slug := buildSlug(title, idx)
	tags := pickTags(faker)
	contentLen := faker.Number(500, 2000)
	content, contentTerms := buildContent(faker, contentLen)

	keywords := make([]string, 0, len(titleTerms)+len(contentTerms)+len(tags)+6)
	keywords = append(keywords, titleTerms...)
	keywords = append(keywords, contentTerms...)
	keywords = append(keywords, tags...)
	if len(tags) > 0 {
		keywords = append(keywords, strings.ToLower(tags[0]))
	}
	keywords = dedupStrings(keywords)

	return generatedPost{
		Post: Post{
			Title:   title,
			Slug:    slug,
			Content: content,
			Tags:    tags,
		},
		Keywords: keywords,
	}
}

func buildTitle(faker *gofakeit.Faker) (string, []string) {
	target := faker.Number(10, 50)
	tokens := make([]string, 0, 10)
	terms := make([]string, 0, 8)
	currentLen := 0

	for currentLen < target {
		var token string
		switch faker.Number(1, 10) {
		case 1, 2, 3, 4, 5, 6:
			token = chineseTerms[faker.Number(0, len(chineseTerms)-1)]
		case 7, 8:
			token = englishTerms[faker.Number(0, len(englishTerms)-1)]
		default:
			token = faker.BuzzWord()
		}
		tokens = append(tokens, token)
		terms = append(terms, normalizeKeyword(token))
		currentLen += runeLength(token)
		if faker.Number(1, 10) > 7 {
			tokens = append(tokens, " ")
			currentLen++
		}
	}

	title := trimToRunes(strings.Join(tokens, ""), 50)
	if runeLength(title) < 10 {
		title += " 实践指南"
		title = trimToRunes(title, 50)
	}
	return strings.TrimSpace(title), dedupStrings(terms)
}

func buildContent(faker *gofakeit.Faker, target int) (string, []string) {
	var builder strings.Builder
	terms := make([]string, 0, 32)
	currentLen := 0

	for currentLen < target {
		choice := faker.Number(1, 10)
		switch {
		case choice <= 5:
			chunk, kws := chineseChunk(faker)
			builder.WriteString(chunk)
			terms = append(terms, kws...)
			currentLen += runeLength(chunk)
		case choice <= 8:
			sentence := faker.Sentence(faker.Number(8, 18))
			builder.WriteString(sentence)
			builder.WriteString(" ")
			terms = append(terms, normalizeKeyword(strings.ToLower(faker.RandomString(englishTerms))))
			currentLen += runeLength(sentence) + 1
		default:
			frag := faker.RandomString(specialFragments)
			builder.WriteString(frag)
			builder.WriteString(" ")
			currentLen += runeLength(frag) + 1
		}
		if faker.Number(1, 10) > 7 {
			builder.WriteString("\n")
			currentLen++
		}
	}

	content := trimToRunes(builder.String(), target)
	if runeLength(content) < 500 {
		padding := strings.Repeat("性能分析与容量规划。", 40)
		content = trimToRunes(content+padding, 500)
	}
	return content, dedupStrings(terms)
}

func chineseChunk(faker *gofakeit.Faker) (string, []string) {
	sz := faker.Number(3, 8)
	parts := make([]string, 0, sz+2)
	terms := make([]string, 0, sz)

	for i := 0; i < sz; i++ {
		term := chineseTerms[faker.Number(0, len(chineseTerms)-1)]
		parts = append(parts, term)
		terms = append(terms, normalizeKeyword(term))
		if i != sz-1 {
			switch faker.Number(1, 4) {
			case 1:
				parts = append(parts, "、")
			case 2:
				parts = append(parts, " 与 ")
			case 3:
				parts = append(parts, " 和 ")
			default:
				parts = append(parts, " ")
			}
		}
	}
	end := []string{"。", "！", "？", "；"}[faker.Number(0, 3)]
	return strings.Join(parts, "") + end, terms
}

func pickTags(faker *gofakeit.Faker) []string {
	count := faker.Number(2, 6)
	tags := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(tags) < count {
		tag := tagPool[faker.Number(0, len(tagPool)-1)]
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func buildSlug(title string, idx int) string {
	var builder strings.Builder
	builder.Grow(len(title) + 16)

	prevDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			prevDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '/':
			if !prevDash && builder.Len() > 0 {
				builder.WriteByte('-')
				prevDash = true
			}
		}
	}
	base := strings.Trim(builder.String(), "-")
	if base == "" {
		base = "post"
	}
	return fmt.Sprintf("%s-%d", base, idx)
}

func writeKeywordsCSV(path string, counter map[string]int, target int, seed int64) error {
	items := make([]keywordFreq, 0, len(counter))
	for keyword, count := range counter {
		if keyword == "" {
			continue
		}
		items = append(items, keywordFreq{Keyword: keyword, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Keyword < items[j].Keyword
		}
		return items[i].Count > items[j].Count
	})

	selected := make([]string, 0, target)
	seen := make(map[string]struct{}, target)
	for _, item := range items {
		if len(selected) >= target {
			break
		}
		kw := strings.TrimSpace(item.Keyword)
		if kw == "" {
			continue
		}
		if _, ok := seen[kw]; ok {
			continue
		}
		seen[kw] = struct{}{}
		selected = append(selected, kw)
	}

	faker := gofakeit.New(seed)
	suffix := 1
	for len(selected) < target {
		candidate := normalizeKeyword(faker.RandomString(chineseTerms))
		if faker.Bool() {
			candidate = normalizeKeyword(faker.RandomString(englishTerms))
		}
		if faker.Number(1, 10) > 7 {
			candidate = normalizeKeyword(fmt.Sprintf("%s-%s-%d", faker.Word(), faker.BuzzWord(), suffix))
			suffix++
		}
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		selected = append(selected, candidate)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建关键词 CSV 失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"keyword"}); err != nil {
		return fmt.Errorf("写入关键词 CSV 头失败: %w", err)
	}
	for _, kw := range selected {
		if err := writer.Write([]string{kw}); err != nil {
			return fmt.Errorf("写入关键词 CSV 内容失败: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("落盘关键词 CSV 失败: %w", err)
	}
	return nil
}

func dedupStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func trimToRunes(s string, maxLen int) string {
	if maxLen <= 0 || runeLength(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}

func runeLength(s string) int {
	return utf8.RuneCountInString(s)
}

func normalizeKeyword(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, ".,;:!?[](){}'\"`")
	return s
}

func runImportAndRebuild(args []string) error {
	var (
		inputPath              string
		dsn                    string
		batchSize              int
		truncateFirst          bool
		skipRebuild            bool
		rebuildURL             string
		rebuildStartTimeoutSec int
		rebuildPollIntervalSec int
		rebuildWaitTimeoutSec  int
		resetFirst             bool
		includeNonPublished    bool
		categoriesRaw          string
	)

	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.StringVar(&inputPath, "input", "posts.jsonl", "文章输入文件（JSONL）")
	fs.StringVar(&dsn, "dsn", "", "MariaDB DSN，例如 root:pwd@tcp(127.0.0.1:3306)/gopalette?parseTime=True&loc=Local")
	fs.IntVar(&batchSize, "batch-size", defaultBatchSize, "单次事务导入条数")
	fs.BoolVar(&truncateFirst, "truncate-first", false, "导入前清空 posts/tags/categories/post_tags/post_likes")
	fs.BoolVar(&skipRebuild, "skip-rebuild", false, "仅导入，不触发索引重建")
	fs.StringVar(&rebuildURL, "rebuild-url", "http://127.0.0.1:8000/v1/search/rebuild", "重建索引 HTTP 地址")
	fs.IntVar(&rebuildStartTimeoutSec, "rebuild-start-timeout-sec", 30, "触发重建接口超时（秒）")
	fs.IntVar(&rebuildPollIntervalSec, "rebuild-poll-interval-sec", 3, "重建状态轮询间隔（秒）")
	fs.IntVar(&rebuildWaitTimeoutSec, "rebuild-wait-timeout-sec", 7200, "等待重建完成超时（秒）")
	fs.BoolVar(&resetFirst, "reset-first", true, "重建索引前先清空索引")
	fs.BoolVar(&includeNonPublished, "include-non-published", false, "重建时包含未发布文章")
	fs.StringVar(&categoriesRaw, "categories", strings.Join(defaultCategories, ","), "分类列表，逗号分隔")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if dsn == "" {
		dsn = os.Getenv("GP_POST_DSN")
	}
	if dsn == "" {
		return fmt.Errorf("缺少 dsn：请传 -dsn 或设置环境变量 GP_POST_DSN")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	categories := parseCategories(categoriesRaw)
	if len(categories) == 0 {
		return fmt.Errorf("分类列表不能为空")
	}

	imported, err := importPostsFromJSONL(inputPath, dsn, batchSize, truncateFirst, categories)
	if err != nil {
		return err
	}
	log.Printf("数据库导入完成，共处理 %d 篇文章", imported)

	if skipRebuild {
		log.Printf("已跳过索引重建（skip-rebuild=true）")
		return nil
	}
	if rebuildPollIntervalSec <= 0 {
		rebuildPollIntervalSec = 3
	}
	if rebuildWaitTimeoutSec <= 0 {
		rebuildWaitTimeoutSec = 7200
	}

	statusURL := strings.TrimRight(rebuildURL, "/") + "/status"
	taskID, err := startRebuildIndex(rebuildURL, resetFirst, includeNonPublished, time.Duration(rebuildStartTimeoutSec)*time.Second)
	if err != nil {
		return err
	}
	log.Printf("已触发异步重建任务 task_id=%s，开始轮询状态...", taskID)

	finalStatus, err := waitRebuildComplete(
		statusURL,
		taskID,
		time.Duration(rebuildPollIntervalSec)*time.Second,
		time.Duration(rebuildWaitTimeoutSec)*time.Second,
	)
	if err != nil {
		return err
	}
	log.Printf("索引重建完成 task_id=%s status=%s indexed=%d/%d", finalStatus.TaskID, finalStatus.Status, finalStatus.IndexedCount, finalStatus.Total)
	return nil
}

func parseCategories(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		result = append(result, v)
	}
	return result
}

func importPostsFromJSONL(inputPath, dsn string, batchSize int, truncateFirst bool, categories []string) (int64, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		return 0, fmt.Errorf("数据库不可用: %w", err)
	}
	if err := ensureRequiredTables(db); err != nil {
		return 0, err
	}
	if truncateFirst {
		if err := truncateImportTables(db); err != nil {
			return 0, err
		}
	}

	categoryIDs, err := ensureCategories(db, categories)
	if err != nil {
		return 0, err
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return 0, fmt.Errorf("打开输入文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	buffer := make([]Post, 0, batchSize)
	var imported int64
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var p Post
		if err := json.Unmarshal(line, &p); err != nil {
			return imported, fmt.Errorf("解析 JSONL 行失败: %w", err)
		}
		if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Slug) == "" {
			continue
		}
		buffer = append(buffer, p)
		if len(buffer) >= batchSize {
			n, err := importBatch(db, buffer, categoryIDs)
			imported += n
			if err != nil {
				return imported, err
			}
			log.Printf("已导入 %d 篇文章", imported)
			buffer = buffer[:0]
		}
	}
	if err := scanner.Err(); err != nil {
		return imported, fmt.Errorf("读取输入文件失败: %w", err)
	}
	if len(buffer) > 0 {
		n, err := importBatch(db, buffer, categoryIDs)
		imported += n
		if err != nil {
			return imported, err
		}
	}

	if err := refreshCounterColumns(db); err != nil {
		return imported, err
	}
	return imported, nil
}

func ensureRequiredTables(db *sql.DB) error {
	required := []string{"posts", "categories", "tags", "post_tags"}
	for _, table := range required {
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&exists); err != nil {
			return fmt.Errorf("检查表 %s 失败: %w", table, err)
		}
		if exists == 0 {
			return fmt.Errorf("缺少表 %s，请先启动 post service 完成 AutoMigrate", table)
		}
	}
	return nil
}

func truncateImportTables(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始清空事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("关闭外键检查失败: %w", err)
	}
	for _, stmt := range []string{
		"TRUNCATE TABLE post_tags",
		"TRUNCATE TABLE post_likes",
		"TRUNCATE TABLE posts",
		"TRUNCATE TABLE tags",
		"TRUNCATE TABLE categories",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("执行 %q 失败: %w", stmt, err)
		}
	}
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return fmt.Errorf("开启外键检查失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交清空事务失败: %w", err)
	}
	return nil
}

func ensureCategories(db *sql.DB, categories []string) ([]int64, error) {
	now := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("初始化分类事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO categories (name, slug, description, post_count, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			name = VALUES(name),
			description = VALUES(description),
			updated_at = VALUES(updated_at),
			deleted_at = NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("准备分类 upsert 失败: %w", err)
	}
	defer stmt.Close()

	ids := make([]int64, 0, len(categories))
	for i, name := range categories {
		slug := buildSlug(name, i+1)
		if !strings.Contains(slug, "-") {
			slug = fmt.Sprintf("category-%d", i+1)
		}
		if _, err := stmt.Exec(name, slug, fmt.Sprintf("%s 相关文章", name), now, now); err != nil {
			return nil, fmt.Errorf("写入分类 %q 失败: %w", name, err)
		}
		var id int64
		if err := tx.QueryRow("SELECT LAST_INSERT_ID()").Scan(&id); err != nil {
			return nil, fmt.Errorf("读取分类 %q id 失败: %w", name, err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交分类事务失败: %w", err)
	}
	return ids, nil
}

func importBatch(db *sql.DB, posts []Post, categoryIDs []int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开始导入事务失败: %w", err)
	}
	defer tx.Rollback()

	postStmt, err := tx.Prepare(`
		INSERT INTO posts
			(title, summary, content, original_content, slug, status, view_count, like_count, comment_count, author_id, category_id, created_at, updated_at)
		VALUES
			(?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			title = VALUES(title),
			summary = VALUES(summary),
			content = VALUES(content),
			original_content = VALUES(original_content),
			status = VALUES(status),
			view_count = VALUES(view_count),
			like_count = VALUES(like_count),
			comment_count = VALUES(comment_count),
			author_id = VALUES(author_id),
			category_id = VALUES(category_id),
			updated_at = VALUES(updated_at),
			deleted_at = NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("准备 posts upsert 失败: %w", err)
	}
	defer postStmt.Close()

	tagStmt, err := tx.Prepare(`
		INSERT INTO tags (name, slug, post_count, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			name = VALUES(name),
			updated_at = VALUES(updated_at),
			deleted_at = NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("准备 tags upsert 失败: %w", err)
	}
	defer tagStmt.Close()

	clearRelStmt, err := tx.Prepare(`DELETE FROM post_tags WHERE post_id = ?`)
	if err != nil {
		return 0, fmt.Errorf("准备清理 post_tags 失败: %w", err)
	}
	defer clearRelStmt.Close()

	relStmt, err := tx.Prepare(`INSERT IGNORE INTO post_tags (post_id, tag_id) VALUES (?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("准备写入 post_tags 失败: %w", err)
	}
	defer relStmt.Close()

	tagIDCache := make(map[string]int64, 256)
	now := time.Now()
	for _, p := range posts {
		hash := stableHash(p.Slug)
		categoryID := categoryIDs[int(hash%uint32(len(categoryIDs)))]
		createdAt := now.Add(-time.Duration(hash%(3600*24*730)) * time.Second)
		updatedAt := createdAt.Add(time.Duration(hash%7200) * time.Second)
		summary := trimToRunes(p.Content, 100)
		if strings.TrimSpace(summary) == "" {
			summary = trimToRunes(p.Title, 100)
		}

		res, err := postStmt.Exec(
			p.Title,
			summary,
			p.Content,
			p.Content,
			p.Slug,
			int64(hash%50000),
			int64(hash%3000),
			int64(hash%2000),
			int64(hash%10000)+1,
			categoryID,
			createdAt,
			updatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("写入文章 slug=%s 失败: %w", p.Slug, err)
		}
		postID, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("获取文章 slug=%s 的 id 失败: %w", p.Slug, err)
		}

		if _, err := clearRelStmt.Exec(postID); err != nil {
			return 0, fmt.Errorf("清理文章标签关系失败(post_id=%d): %w", postID, err)
		}

		for _, tagName := range dedupStrings(p.Tags) {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagSlug := buildSlug(tagName, int(hash%100000)+1)
			if !strings.Contains(tagSlug, "-") {
				tagSlug = "tag-" + strconv.FormatUint(uint64(stableHash(tagName)), 10)
			}

			tagID, ok := tagIDCache[tagName]
			if !ok {
				if _, err := tagStmt.Exec(tagName, tagSlug, now, now); err != nil {
					return 0, fmt.Errorf("写入标签 %q 失败: %w", tagName, err)
				}
				if err := tx.QueryRow("SELECT LAST_INSERT_ID()").Scan(&tagID); err != nil {
					return 0, fmt.Errorf("读取标签 %q id 失败: %w", tagName, err)
				}
				tagIDCache[tagName] = tagID
			}
			if _, err := relStmt.Exec(postID, tagID); err != nil {
				return 0, fmt.Errorf("写入 post_tags 失败(post_id=%d, tag_id=%d): %w", postID, tagID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交导入事务失败: %w", err)
	}
	return int64(len(posts)), nil
}

func refreshCounterColumns(db *sql.DB) error {
	if _, err := db.Exec(`
		UPDATE categories c
		SET c.post_count = (
			SELECT COUNT(*)
			FROM posts p
			WHERE p.category_id = c.id AND p.deleted_at IS NULL
		)
	`); err != nil {
		return fmt.Errorf("更新 categories.post_count 失败: %w", err)
	}
	if _, err := db.Exec(`
		UPDATE tags t
		SET t.post_count = (
			SELECT COUNT(*)
			FROM post_tags pt
			INNER JOIN posts p ON p.id = pt.post_id
			WHERE pt.tag_id = t.id AND p.deleted_at IS NULL
		)
	`); err != nil {
		return fmt.Errorf("更新 tags.post_count 失败: %w", err)
	}
	return nil
}

func stableHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if h == 0 {
		return 1
	}
	return h
}

type rebuildTaskResponse struct {
	TaskID       string `json:"task_id"`
	Status       string `json:"status"`
	IndexedCount int64  `json:"indexed_count"`
	Total        int64  `json:"total"`
	ErrorMessage string `json:"error_message"`
}

func startRebuildIndex(url string, resetFirst, includeNonPublished bool, timeout time.Duration) (string, error) {
	payload := map[string]bool{
		"reset_first":           resetFirst,
		"include_non_published": includeNonPublished,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化重建请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建重建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用重建接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取重建响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("重建接口返回异常状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", fmt.Errorf("解析重建响应失败: %w", err)
	}

	if taskMap, ok := raw["task"].(map[string]any); ok {
		if taskID, ok := taskMap["task_id"].(string); ok && taskID != "" {
			return taskID, nil
		}
		if taskID, ok := taskMap["taskId"].(string); ok && taskID != "" {
			return taskID, nil
		}
	}
	if taskID, ok := raw["task_id"].(string); ok && taskID != "" {
		return taskID, nil
	}
	return "", fmt.Errorf("重建接口未返回 task_id: %s", string(respBody))
}

func waitRebuildComplete(statusURL, taskID string, pollInterval, waitTimeout time.Duration) (*rebuildTaskResponse, error) {
	deadline := time.Now().Add(waitTimeout)
	client := &http.Client{Timeout: 30 * time.Second}

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("等待重建任务超时 task_id=%s", taskID)
		}

		status, err := fetchRebuildStatus(client, statusURL, taskID)
		if err != nil {
			return nil, err
		}
		log.Printf("重建任务进度 task_id=%s status=%s indexed=%d/%d", status.TaskID, status.Status, status.IndexedCount, status.Total)
		switch status.Status {
		case "SUCCEEDED":
			return status, nil
		case "FAILED":
			return nil, fmt.Errorf("重建任务失败 task_id=%s: %s", status.TaskID, status.ErrorMessage)
		}

		time.Sleep(pollInterval)
	}
}

func fetchRebuildStatus(client *http.Client, statusURL, taskID string) (*rebuildTaskResponse, error) {
	reqURL := strings.TrimRight(statusURL, "/")
	if taskID != "" {
		reqURL += "/" + taskID
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建状态请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用状态接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取状态响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("状态接口返回异常状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w", err)
	}

	taskRaw, ok := raw["task"].(map[string]any)
	if !ok {
		taskRaw = raw
	}

	status := &rebuildTaskResponse{
		TaskID:       parseString(taskRaw, "task_id", "taskId"),
		Status:       strings.ToUpper(parseString(taskRaw, "status")),
		IndexedCount: parseInt64(taskRaw, "indexed_count", "indexedCount"),
		Total:        parseInt64(taskRaw, "total"),
		ErrorMessage: parseString(taskRaw, "error_message", "errorMessage"),
	}
	if status.TaskID == "" {
		status.TaskID = taskID
	}
	return status, nil
}

func parseString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseInt64(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := m[key].(float64); ok {
			return int64(v)
		}
		if v, ok := m[key].(int64); ok {
			return v
		}
	}
	return 0
}
