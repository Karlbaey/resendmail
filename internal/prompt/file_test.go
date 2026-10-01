package prompt

import (
	"reflect"
	"testing"
)

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name   string
		files  []string
		picked string
		want   []string
	}{
		{
			name:   "empty picked is ignored",
			files:  []string{"a.txt"},
			picked: "",
			want:   []string{"a.txt"},
		},
		{
			name:   "duplicate is ignored",
			files:  []string{"a.txt", "b.txt"},
			picked: "a.txt",
			want:   []string{"a.txt", "b.txt"},
		},
		{
			name:   "new file is appended",
			files:  []string{"a.txt"},
			picked: "b.txt",
			want:   []string{"a.txt", "b.txt"},
		},
		{
			name:   "appending to empty list",
			files:  nil,
			picked: "a.txt",
			want:   []string{"a.txt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUnique(tt.files, tt.picked)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("appendUnique(%v, %q) = %v, want %v", tt.files, tt.picked, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		array []string
		s     string
		want  bool
	}{
		{array: nil, s: "x", want: false},
		{array: []string{"a", "b"}, s: "a", want: true},
		{array: []string{"a", "b"}, s: "c", want: false},
		{array: []string{"a", "a"}, s: "a", want: true},
	}
	for _, tt := range tests {
		got := contains(tt.array, tt.s)
		if got != tt.want {
			t.Fatalf("contains(%v, %q) = %v, want %v", tt.array, tt.s, got, tt.want)
		}
	}
}
