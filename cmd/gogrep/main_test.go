package main

import "testing"

func TestExecute(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "version", args: []string{"--version"}, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := execute(test.args); got != test.want {
				t.Fatalf("execute() = %d, want %d", got, test.want)
			}
		})
	}
}
