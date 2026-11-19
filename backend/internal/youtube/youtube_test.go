package youtube

import "testing"

func TestExtractID(t *testing.T) {
	ok := map[string]string{
		"https://www.youtube.com/watch?v=jfKfPfyJRdk":            "jfKfPfyJRdk",
		"https://youtu.be/jfKfPfyJRdk?si=abc":                    "jfKfPfyJRdk",
		"youtube.com/watch?v=jfKfPfyJRdk&list=RD":                "jfKfPfyJRdk",
		"https://m.youtube.com/watch?v=jfKfPfyJRdk":              "jfKfPfyJRdk",
		"https://www.youtube.com/embed/jfKfPfyJRdk":              "jfKfPfyJRdk",
		"https://www.youtube.com/live/jfKfPfyJRdk?feature=share": "jfKfPfyJRdk",
		"https://music.youtube.com/watch?v=jfKfPfyJRdk":          "jfKfPfyJRdk",
		"jfKfPfyJRdk": "jfKfPfyJRdk",
	}
	for in, want := range ok {
		got, err := ExtractID(in)
		if err != nil || got != want {
			t.Errorf("ExtractID(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "https://vimeo.com/123", "https://youtube.com/watch?v=short", "hello world"} {
		if _, err := ExtractID(in); err == nil {
			t.Errorf("ExtractID(%q) expected error", in)
		}
	}
}
