package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func Test(ctx context.Context, policy interfaces.PolicyClient, testDataSet *model.TestDataSet) []error {
	var errs []error
	for schema, dataSets := range testDataSet.Detect {
		for filename, data := range dataSets {
			var resp model.PolicyResult
			if err := policy.Query(ctx, "data.alert."+schema, data, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					errs = append(errs, goerr.Wrap(err, "should be detected, but not detected", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename), goerr.T(model.ErrTagTestFailed)))
				} else {
					errs = append(errs, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename)))
				}
			}
		}
	}

	for schema, dataSets := range testDataSet.Ignore {
		for filename, data := range dataSets {
			var resp model.PolicyResult
			if err := policy.Query(ctx, "data.alert."+schema, data, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					continue
				}

				errs = append(errs, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename)))
			}

			if len(resp.Alert) > 0 {
				errs = append(errs, goerr.New("should be ignored, but detected", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename), goerr.T(model.ErrTagTestFailed)))
			}
		}
	}

	return errs
}

func hashPolicyData(data map[string]string) string { // Sort keys and create deterministic JSON bytes
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	orderedData := make(map[string]string)
	for _, k := range keys {
		orderedData[k] = data[k]
	}

	jsonBytes, err := json.Marshal(orderedData)
	if err != nil {
		logging.Default().Error("failed to marshal policy data", "error", err, "data", orderedData)
		panic(err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

type PolicyData map[string]string

type Factory func(data PolicyData) (interfaces.PolicyClient, error)

type Service struct {
	repo         interfaces.Repository
	policyClient interfaces.PolicyClient
	testData     *model.TestDataSet
	baseHash     string
	factory      Factory
}

func (s *Service) TestData() *model.TestDataSet {
	newTestData := &model.TestDataSet{
		Detect: make(model.TestData),
		Ignore: make(model.TestData),
	}

	for schema, dataSets := range s.testData.Detect {
		newTestData.Detect[schema] = make(map[string]any)
		for filename, data := range dataSets {
			newTestData.Detect[schema][filename] = data
		}
	}

	for schema, dataSets := range s.testData.Ignore {
		newTestData.Ignore[schema] = make(map[string]any)
		for filename, data := range dataSets {
			newTestData.Ignore[schema][filename] = data
		}
	}

	return newTestData
}

func (s *Service) Clone(policyClient interfaces.PolicyClient, testData *model.TestDataSet) *Service {
	newSvc := &Service{
		repo:         s.repo,
		policyClient: policyClient,
		testData:     testData,
		baseHash:     hashPolicyData(policyClient.Sources()),
		factory:      s.factory,
	}

	return newSvc
}

func (s *Service) Test(ctx context.Context) []error {
	return Test(ctx, s.policyClient, s.testData)
}

func New(repo interfaces.Repository, policyClient interfaces.PolicyClient, testData *model.TestDataSet, opts ...Option) *Service {
	svc := &Service{
		repo:         repo,
		policyClient: policyClient,
		testData:     testData,
		baseHash:     hashPolicyData(policyClient.Sources()),
		factory: func(data PolicyData) (interfaces.PolicyClient, error) {
			return opaq.New(opaq.DataMap(data))
		},
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

type Option func(*Service)

func WithFactory(factory Factory) Option {
	return func(s *Service) {
		s.factory = factory
	}
}

func (s *Service) NewClient(ctx context.Context) (interfaces.PolicyClient, error) {
	policyData := s.policyClient.Sources()

	// Get existing policy hash from repository
	existingPolicy, err := s.repo.GetPolicy(ctx, s.baseHash)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get policy from repository", goerr.V("baseHash", s.baseHash))
	}

	// If policy doesn't exist or hash is different, save the new policy
	if existingPolicy == nil {
		newPolicy := &model.PolicyData{
			Hash:      s.baseHash,
			Data:      policyData,
			CreatedAt: time.Now(),
		}
		if err := s.repo.SavePolicy(ctx, newPolicy); err != nil {
			return nil, goerr.Wrap(err, "failed to save policy", goerr.V("baseHash", s.baseHash), goerr.V("policyData", policyData))
		}

		return s.policyClient, nil
	}

	newClient, err := s.factory(existingPolicy.Data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create new policy client", goerr.V("baseHash", s.baseHash))
	}

	return newClient, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, data map[string]string) error {
	policyData := &model.PolicyData{
		Hash:      s.baseHash,
		Data:      data,
		CreatedAt: clock.Now(ctx),
	}

	if err := s.repo.SavePolicy(ctx, policyData); err != nil {
		return goerr.Wrap(err, "failed to save policy", goerr.V("hash", s.baseHash), goerr.V("data", data))
	}

	return nil
}
