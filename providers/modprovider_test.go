package providers

import (
	"testing"

	cfSchema "github.com/sjet47/go-curseforge/schema"
)

func TestCurseForgeLogoURL(t *testing.T) {
	tests := []struct {
		name string
		logo cfSchema.ModAsset
		want string
	}{
		{
			name: "uses thumbnail url first",
			logo: cfSchema.ModAsset{
				ThumbnailUrl: " https://example.com/thumb.png ",
				URL:          "https://example.com/original.webp",
			},
			want: "https://example.com/thumb.png",
		},
		{
			name: "falls back to url",
			logo: cfSchema.ModAsset{
				URL: " https://media.forgecdn.net/avatars/1025/127/638548475358792693.webp ",
			},
			want: "https://media.forgecdn.net/avatars/1025/127/638548475358792693.webp",
		},
		{
			name: "returns empty when both are empty",
			logo: cfSchema.ModAsset{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := curseForgeLogoURL(tt.logo)
			if got != tt.want {
				t.Fatalf("curseForgeLogoURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
