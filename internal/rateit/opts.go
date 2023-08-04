package rateit

type Option func(*Client)

func WithUpstream(upstream string) Option {
	return func(client *Client) {
		if upstream == "" {
			return
		}

		client.httpc.SetBaseURL(upstream)
	}
}

func WithVerbose(verbose bool) Option {
	return func(client *Client) {
		client.httpc.SetDebug(verbose)
	}
}
