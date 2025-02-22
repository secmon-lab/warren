package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

	errs = append(errs, runTest(ctx, policy, testDataSet.Detect.Data, true)...)
	errs = append(errs, runTest(ctx, policy, testDataSet.Ignore.Data, false)...)

	return errs
}

func runTest(ctx context.Context, policy interfaces.PolicyClient, dataSets map[string]map[string]any, shouldDetect bool) []error {
	var errs []error
	for schema, dataSet := range dataSets {
		for filename, testData := range dataSet {
			var resp model.PolicyResult
			if err := policy.Query(ctx, "data.alert."+schema, testData, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					if shouldDetect {
						errs = append(errs, goerr.Wrap(err, "should be detected, but not detected",
							goerr.V("schema", schema),
							goerr.V("filename", filename),
							goerr.T(model.ErrTagTestFailed)))
					}
					continue
				}
				if len(resp.Alert) == 0 && shouldDetect {
					errs = append(errs, goerr.Wrap(err, "should be detected, but not detected",
						goerr.V("schema", schema),
						goerr.V("filename", filename),
						goerr.T(model.ErrTagTestFailed)))
					continue
				}

				if len(resp.Alert) > 0 && !shouldDetect {
					errs = append(errs, goerr.Wrap(err, "should be ignored, but detected",
						goerr.V("schema", schema),
						goerr.V("filename", filename),
						goerr.T(model.ErrTagTestFailed)))
				}
			}

			if !shouldDetect && len(resp.Alert) > 0 {
				errs = append(errs, goerr.New("should be ignored, but detected",
					goerr.V("schema", schema),
					goerr.V("filename", filename),
					goerr.T(model.ErrTagTestFailed)))
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

func (s *Service) Sources() map[string]string {
	return s.policyClient.Sources()
}

func (s *Service) TestDataSet() *model.TestDataSet {
	newTestData := &model.TestDataSet{
		Detect: s.testData.Detect.Clone(),
		Ignore: s.testData.Ignore.Clone(),
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

func (s *Service) Save(ctx context.Context, rootDir string) error {
	logger := logging.From(ctx)
	sources := s.policyClient.Sources()

	for filename, source := range sources {
		path := filepath.Join(rootDir, filename)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return goerr.Wrap(err, "failed to create directory", goerr.V("path", path))
		}

		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			return goerr.Wrap(err, "failed to save policy", goerr.V("filename", filename))
		}

		logger.Debug("saved policy", "file", path)
	}

	testDataSet := s.TestDataSet()
	saveTestData := func(d *model.TestData) error {
		for schema, dataSets := range d.Data {
			for filename, testData := range dataSets {
				jsonData, err := json.MarshalIndent(testData, "", "  ")
				if err != nil {
					return goerr.Wrap(err, "failed to marshal test data", goerr.V("schema", schema), goerr.V("filename", filename))
				}

				path := filepath.Join(rootDir, d.BasePath, schema, filename)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					return goerr.Wrap(err, "failed to create directory", goerr.V("path", path))
				}

				if err := os.WriteFile(path, jsonData, 0644); err != nil {
					return goerr.Wrap(err, "failed to save test data", goerr.V("schema", schema), goerr.V("filename", filename))
				}

				logger.Debug("saved test data", "file", path)
			}
		}
		return nil
	}

	if err := saveTestData(testDataSet.Detect); err != nil {
		return err
	}

	if err := saveTestData(testDataSet.Ignore); err != nil {
		return err
	}

	return nil
}
