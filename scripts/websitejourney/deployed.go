package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const CanonicalDeployedOrigin = "https://backstop.sh"

type DeployedRequest struct {
	Origin          string
	Commit          string
	RunID           string
	FollowRedirects bool
	ClaimRollback   bool
	DirectLoad      bool
	ScreenshotOnly  bool
	Capability      string
	Fetch           DeployedFetcher
}

type DeployedFetcher func(rawURL string) (status int, body string, err error)

type DeployedIdentityMutation struct {
	Name            string `yaml:"name"`
	Origin          string `yaml:"origin"`
	FollowRedirects bool   `yaml:"follow_redirects"`
	StaleCommit     bool   `yaml:"stale_commit"`
	StaleRun        bool   `yaml:"stale_run"`
	MissingMarker   bool   `yaml:"missing_marker"`
	MissingRoute    string `yaml:"missing_route"`
	ClaimRollback   bool   `yaml:"claim_rollback"`
	ScreenshotOnly  bool   `yaml:"screenshot_only"`
	DirectLoad      bool   `yaml:"direct_load"`
	ExpectedError   string `yaml:"expected_error"`
}

var (
	deploymentMarkerPattern = regexp.MustCompile(`<meta name="backstop-deployment" content="commit=([0-9a-f]{40});run=([1-9][0-9]*)">`)
	fullCommitPattern       = regexp.MustCompile(`^[0-9a-f]{40}$`)
	runIDPattern            = regexp.MustCompile(`^[1-9][0-9]*$`)
)

func TraverseDeployedSite(m WebsiteCapabilityMap, req DeployedRequest) error {
	if req.ClaimRollback {
		return fmt.Errorf("deployed: rollback claims are prohibited")
	}
	if req.FollowRedirects {
		return fmt.Errorf("deployed: redirect following is prohibited")
	}
	if req.Origin != CanonicalDeployedOrigin {
		return fmt.Errorf("deployed: origin %q is not %s", req.Origin, CanonicalDeployedOrigin)
	}
	if !fullCommitSHA(req.Commit) || !validRunID(req.RunID) {
		return fmt.Errorf("deployed: commit/run identity is missing or malformed")
	}
	if req.Fetch == nil {
		req.Fetch = DefaultDeployedFetcher()
	}
	documents := map[string]string{}
	for _, route := range CanonicalBuiltRoutes() {
		status, body, err := req.Fetch(req.Origin + route)
		if err != nil {
			return fmt.Errorf("deployed: %s: %w", route, err)
		}
		if status != 200 {
			return fmt.Errorf("deployed: %s: HTTP %d, want 200", route, status)
		}
		if !HasDeployedIdentity(body, req.Commit, req.RunID) {
			return fmt.Errorf("deployed: %s: stale or missing deployment marker for %s run %s", route, req.Commit, req.RunID)
		}
		documents[route] = body
	}
	if err := AssertNoPublishedRuntime(documents); err != nil {
		return err
	}
	opts := TraverseOptions{DirectLoad: req.DirectLoad, ScreenshotOnly: req.ScreenshotOnly}
	for _, journey := range m.Journeys {
		if req.Capability != "" && !strings.HasPrefix(journey.GlobalKey, req.Capability+"/") {
			continue
		}
		if err := traverseJourney(documents, journey, opts); err != nil {
			return fmt.Errorf("%s: deployed journey failed: %w", journey.GlobalKey, err)
		}
	}
	return nil
}

func HasDeployedIdentity(document, commit, runID string) bool {
	matches := deploymentMarkerPattern.FindAllStringSubmatch(document, -1)
	if len(matches) != 1 {
		return false
	}
	return matches[0][1] == commit && matches[0][2] == runID
}

func StampDeployedIdentity(builtRoot, commit, runID string) error {
	if !fullCommitSHA(commit) || !validRunID(runID) {
		return fmt.Errorf("deployed: commit/run identity is missing or malformed")
	}
	marker := fmt.Sprintf(`<meta name="backstop-deployment" content="commit=%s;run=%s">`, commit, runID)
	for _, route := range CanonicalBuiltRoutes() {
		path := BuiltRoutePath(builtRoot, route)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(data)
		if !strings.Contains(body, "<head>") {
			body = strings.Replace(body, "<!doctype html><html>", "<!doctype html><html><head></head>", 1)
		}
		if strings.Contains(body, `name="backstop-deployment"`) {
			return fmt.Errorf("deployed: %s: deployment marker already present", route)
		}
		body = strings.Replace(body, "</head>", marker+"</head>", 1)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func FixtureDeployedFetcher(builtRoot, origin string, missingRoute string) DeployedFetcher {
	return func(rawURL string) (int, string, error) {
		if !strings.HasPrefix(rawURL, origin) {
			return 0, "", fmt.Errorf("deployed: non-canonical URL %s", rawURL)
		}
		route := strings.TrimPrefix(rawURL, origin)
		if route == "" {
			route = "/"
		}
		if !strings.HasSuffix(route, "/") && route != "/" {
			route += "/"
		}
		if missingRoute != "" && route == missingRoute {
			return 404, "", nil
		}
		data, err := os.ReadFile(BuiltRoutePath(builtRoot, route))
		if err != nil {
			return 404, "", nil
		}
		return 200, string(data), nil
	}
}

func DefaultDeployedFetcher() DeployedFetcher {
	return DeployedFetcherWithClient(&http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return fmt.Errorf("deployed: redirect following is prohibited")
		},
	})
}

func DeployedFetcherWithClient(client *http.Client) DeployedFetcher {
	return func(rawURL string) (int, string, error) {
		if !strings.HasPrefix(rawURL, CanonicalDeployedOrigin) {
			return 0, "", fmt.Errorf("deployed: origin %q is not %s", rawURL, CanonicalDeployedOrigin)
		}
		resp, err := client.Get(rawURL)
		if err != nil {
			if resp != nil {
				if closeErr := resp.Body.Close(); closeErr != nil {
					return 0, "", fmt.Errorf("%w: %w", err, closeErr)
				}
			}
			return 0, "", err
		}
		body, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil {
			return resp.StatusCode, "", err
		}
		if closeErr != nil {
			return resp.StatusCode, "", closeErr
		}
		return resp.StatusCode, string(body), nil
	}
}

func ApplyDeployedIdentityMutation(builtRoot, commit, runID string, mutation DeployedIdentityMutation) (DeployedRequest, error) {
	req := DeployedRequest{
		Origin:          CanonicalDeployedOrigin,
		Commit:          commit,
		RunID:           runID,
		FollowRedirects: mutation.FollowRedirects,
		ClaimRollback:   mutation.ClaimRollback,
		DirectLoad:      mutation.DirectLoad,
		ScreenshotOnly:  mutation.ScreenshotOnly,
	}
	if mutation.Origin != "" {
		req.Origin = mutation.Origin
	}
	if err := StampDeployedIdentity(builtRoot, commit, runID); err != nil {
		return req, err
	}
	if mutation.StaleCommit || mutation.StaleRun || mutation.MissingMarker {
		path := BuiltRoutePath(builtRoot, "/")
		data, err := os.ReadFile(path)
		if err != nil {
			return req, err
		}
		body := string(data)
		switch {
		case mutation.MissingMarker:
			body = deploymentMarkerPattern.ReplaceAllString(body, "")
		case mutation.StaleCommit:
			body = strings.Replace(body, commit, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 1)
		case mutation.StaleRun:
			body = strings.Replace(body, "run="+runID, "run=9"+runID, 1)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return req, err
		}
	}
	req.Fetch = FixtureDeployedFetcher(builtRoot, CanonicalDeployedOrigin, mutation.MissingRoute)
	return req, nil
}

func fullCommitSHA(commit string) bool {
	return fullCommitPattern.MatchString(commit)
}

func validRunID(runID string) bool {
	return runIDPattern.MatchString(runID)
}
