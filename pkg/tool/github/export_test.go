package github

import "net/http"

// GraphQLBlameResponse is exported for testing
type GraphQLBlameResponse = graphQLResponse

// GraphQLBlameData is exported for testing
type GraphQLBlameData = graphQLBlameData

// GraphQLRepository is exported for testing
type GraphQLRepository = graphQLRepository

// GraphQLObject is exported for testing
type GraphQLObject = graphQLObject

// GraphQLBlame is exported for testing
type GraphQLBlame = graphQLBlame

// GraphQLBlameRange is exported for testing
type GraphQLBlameRange = graphQLBlameRange

// GraphQLCommitRef is exported for testing
type GraphQLCommitRef = graphQLCommitRef

// GraphQLCommitAuthor is exported for testing
type GraphQLCommitAuthor = graphQLCommitAuthor

// SetHTTPClient sets the HTTP client for testing
func (x *Action) SetHTTPClient(client *http.Client) {
	x.httpClient = client
}

// BuildCodeSearchQuery is exported for testing
var BuildCodeSearchQuery = buildCodeSearchQuery

// BuildIssueSearchQuery is exported for testing
var BuildIssueSearchQuery = buildIssueSearchQuery

// ParseRepoFilter is exported for testing
var ParseRepoFilter = parseRepoFilter
