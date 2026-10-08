package github

import "testing"

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantOwner   string
		wantRepo    string
		wantErr     bool
	}{
		{"https", "https://github.com/owner/repo", "owner", "repo", false},
		{"trailing .git", "https://github.com/owner/repo.git", "owner", "repo", false},
		{"trailing slash", "https://github.com/owner/repo/", "owner", "repo", false},
		{"uppercase host", "https://GitHub.com/Owner/Repo", "Owner", "Repo", false},
		{"empty", "", "", "", true},
		{"wrong host", "https://gitlab.com/owner/repo", "", "", true},
		{"missing repo", "https://github.com/owner", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, r, err := ParseRepoURL(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got (%q,%q)", o, r)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o != c.wantOwner || r != c.wantRepo {
				t.Fatalf("got (%q,%q), want (%q,%q)", o, r, c.wantOwner, c.wantRepo)
			}
		})
	}
}
