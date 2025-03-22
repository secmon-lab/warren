package list_test

/*
func TestService_Run(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		alerts   []alert.Alert
		expected []alert.Alert
		wantErr  bool
	}{
		{
			name: "filter by user",
			args: []string{"user", "<@U123>"},
			alerts: []alert.Alert{
				{Assignee: &slack.SlackUser{ID: "U123"}},
				{Assignee: &slack.SlackUser{ID: "U456"}},
			},
			expected: []alert.Alert{
				{Assignee: &slack.SlackUser{ID: "U123"}},
			},
		},
		{
			name: "sort by created at",
			args: []string{"sort", "created_at"},
			alerts: []alert.Alert{
				{CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			expected: []alert.Alert{
				{CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name:    "invalid command",
			args:    []string{"invalid"},
			wantErr: true,
		},
		{
			name: "limit alerts",
			args: []string{"limit", "1"},
			alerts: []alert.Alert{
				{ID: "1"},
				{ID: "2"},
			},
			expected: []alert.Alert{
				{ID: "1"},
			},
		},
		{
			name: "offset alerts",
			args: []string{"offset", "1"},
			alerts: []alert.Alert{
				{ID: "1"},
				{ID: "2"},
			},
			expected: []alert.Alert{
				{ID: "2"},
			},
		},
		{
			name: "filter by status",
			args: []string{"status", "new", "resolved"},
			alerts: []alert.Alert{
				{Status: alert.StatusNew},
				{Status: alert.StatusAcknowledged},
				{Status: alert.StatusResolved},
			},
			expected: []alert.Alert{
				{Status: alert.StatusNew},
				{Status: alert.StatusResolved},
			},
		},
		{
			name:    "invalid status",
			args:    []string{"status", "invalid"},
			wantErr: true,
		},
		{
			name: "status pipeline",
			args: []string{"status", "new", "|", "status", "resolved"},
			alerts: []alert.Alert{
				{Status: alert.StatusNew},
				{Status: alert.StatusAcknowledged},
				{Status: alert.StatusResolved},
			},
			expected: nil,
		},
		{
			name: "limit offset pipeline",
			args: []string{"limit", "2", "|", "offset", "1"},
			alerts: []alert.Alert{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			expected: []alert.Alert{
				{ID: "2"},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			llmMock := &mock.LLMClientMock{
				SendMessageFunc: func(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
					return nil, goerr.New("test")
				},
			}
			repo := repository.NewMemory()
			svc := list.New(repo, list.WithLLM(llmMock))
			var alertList *alert.List
			th := mock.SlackThreadServiceMock{
				ReplyFunc: func(ctx context.Context, message string) {},
				PostAlertListFunc: func(ctx context.Context, list *alert.List) error {
					alertList = list
					return nil
				},
				ChannelIDFunc: func() string {
					return "C123"
				},
				ThreadIDFunc: func() string {
					return "T123"
				},
			}
			args := append([]string{"|"}, tt.args...)
			err := svc.Run(t.Context(), &th, &slack.SlackUser{}, source.Static(tt.alerts), args)
			if tt.wantErr {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err).Must()
			gt.Equal(t, tt.expected, alertList.Alerts)
		})
	}
}
*/
