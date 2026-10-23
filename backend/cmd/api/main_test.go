package main

import "testing"

func TestListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		httpAddr string
		want     string
	}{
		{
			name: "Render port",
			port: "10000",
			want: "0.0.0.0:10000",
		},
		{
			name:     "Render port takes precedence",
			port:     "10000",
			httpAddr: "127.0.0.1:8080",
			want:     "0.0.0.0:10000",
		},
		{
			name:     "local address",
			httpAddr: "127.0.0.1:9000",
			want:     "127.0.0.1:9000",
		},
		{
			name: "default address",
			want: "127.0.0.1:8080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PORT", test.port)
			t.Setenv("HTTP_ADDR", test.httpAddr)

			if got := listenAddress(); got != test.want {
				t.Fatalf("listenAddress() = %q, want %q", got, test.want)
			}
		})
	}
}
