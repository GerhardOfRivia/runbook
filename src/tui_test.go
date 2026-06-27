package main

import "testing"

func TestHasSudo(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{
			name: "Simple sudo command",
			code: "sudo apt-get update",
			want: true,
		},
		{
			name: "Sudo in pipeline",
			code: "echo 'hello' | sudo tee /etc/test",
			want: true,
		},
		{
			name: "Commented sudo should be ignored",
			code: "# sudo rm -rf /",
			want: false,
		},
		{
			name: "Sudo on second line, first is comment",
			code: "# Run with root\nsudo systemctl restart docker",
			want: true,
		},
		{
			name: "Sudo as substring should be ignored",
			code: "echo pseudocode\nvarsudo=1\nsudo_param=true",
			want: false,
		},
		{
			name: "Sudo inside quotes",
			code: "echo \"sudo command\"",
			want: true,
		},
		{
			name: "Sudo command ending with semicolon",
			code: "sudo apt-get update; echo done",
			want: true,
		},
		{
			name: "Sudo inside shell operators",
			code: "sudo(apt-get)",
			want: true,
		},
		{
			name: "No sudo at all",
			code: "ls -la\necho 'hello world'",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasSudo(tt.code)
			if got != tt.want {
				t.Errorf("hasSudo() = %v, want %v. Code:\n%s", got, tt.want, tt.code)
			}
		})
	}
}
