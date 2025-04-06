package urlscan

/*
type Action struct {
	apiKey  string
	baseURL string
	backoff time.Duration
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "urlscan-api-key",
			Usage:       "URLScan API key",
			Destination: &x.apiKey,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_URLSCAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "urlscan-base-url",
			Usage:       "URLScan API base URL",
			Destination: &x.baseURL,
			Category:    "Action",
			Value:       "https://urlscan.io/api/v1",
			Sources:     cli.EnvVars("WARREN_URLSCAN_BASE_URL"),
		},
		&cli.DurationFlag{
			Name:        "urlscan-backoff",
			Usage:       "URLScan API backoff duration",
			Destination: &x.backoff,
			Category:    "Action",
			Value:       time.Duration(3) * time.Second,
			Sources:     cli.EnvVars("WARREN_URLSCAN_BACKOFF"),
		},
	}
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return errs.ErrActionUnavailable
	}
	if _, err := url.Parse(x.baseURL); err != nil {
		return goerr.Wrap(err, "invalid base URL", goerr.V("base_url", x.baseURL))
	}

	return nil
}

func (x Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
		slog.Duration("backoff", x.backoff),
	)
}

func (x *Action) Spec() action.ActionSpec {
	return action.ActionSpec{
		Name:        "urlscan",
		Description: "Scan a URL with URLScan",
		Args: []action.ArgumentSpec{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to scan",
				Required:    true,
			},
		},
	}
}

func (x *Action) Execute(ctx context.Context, slack interfaces.SlackThreadService, ssn interfaces.LLMSession, args action.Arguments) (*action.ActionResult, error) {
	if err := x.Spec().Validate(args); err != nil {
		return nil, err
	}

	url, ok := args["url"].(string)
	if !ok {
		return nil, goerr.New("url is required")
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", x.baseURL+"/scan", strings.NewReader(`{"url":"`+url+`"}`))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", x.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to scan URL", goerr.V("status_code", resp.StatusCode), goerr.V("body", string(body)))
	}

	var result struct {
		UUID    string `json:"uuid"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, goerr.Wrap(err, "failed to decode response")
	}

	// Poll the result API until scan is complete
	resultURL := fmt.Sprintf("%s/result/%s/", x.baseURL, result.UUID)
	for i := 0; i < 5; i++ { // Try up to 5 times with increasing delay
		time.Sleep(time.Duration(1<<i) * x.backoff)

		req, err := http.NewRequestWithContext(ctx, "GET", resultURL, nil)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create result request")
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get result")
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to read response body")
			}
			return &action.ActionResult{
				Message: "Scan result of " + url,
				Type:    action.ActionResultTypeJSON,
				Data:    string(body),
			}, nil
		case http.StatusNotFound:
			continue
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goerr.New("failed to get scan result", goerr.V("status_code", resp.StatusCode), goerr.V("body", string(body)))
		}
	}

	return nil, goerr.New("failed to get scan result")
}
*/
