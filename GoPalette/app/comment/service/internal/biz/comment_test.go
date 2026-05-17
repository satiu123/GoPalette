package biz

import "testing"

func TestSanitizeCommentContentEscapesHTML(t *testing.T) {
	got := SanitizeCommentContent(` <script>window.__xss_test=1</script>正常评论 `)
	want := `&lt;script&gt;window.__xss_test=1&lt;/script&gt;正常评论`
	if got != want {
		t.Fatalf("SanitizeCommentContent() = %q, want %q", got, want)
	}
}

func TestSanitizeCommentContentIsIdempotent(t *testing.T) {
	content := `&lt;script&gt;alert(1)&lt;/script&gt;`
	got := SanitizeCommentContent(SanitizeCommentContent(content))
	want := `&lt;script&gt;alert(1)&lt;/script&gt;`
	if got != want {
		t.Fatalf("SanitizeCommentContent() = %q, want %q", got, want)
	}
}
