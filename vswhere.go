//+build windows

// Package vswhere implements an interface to Microsoft's vswhere[1], a Visual
// Studio Installation locator. vswhere must be installed, and is assumed to
// be present in "%ProgramFiles(x86)%\Microsoft Visual Studio\Installer\vswhere.exe".
//
//   [1]: https://github.com/microsoft/vswhere
package vswhere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Installation is an individual installation of Visual Studio.
type Installation struct {
	InstanceID          string     `json:"instanceId"`
	InstallDate         time.Time  `json:"installDate"`
	InstallationName    string     `json:"installationName"`
	InstallationPath    string     `json:"installationPath"`
	InstallationVersion string     `json:"installationVersion"`
	ProductID           string     `json:"productId"`
	ProductPath         string     `json:"productPath"`
	State               uint64     `json:"state"`
	IsComplete          bool       `json:"isComplete"`
	IsLaunchable        bool       `json:"isLaunchable"`
	IsPrerelease        bool       `json:"isPrerelease"`
	IsRebootRequired    bool       `json:"isRebootRequired"`
	DisplayName         string     `json:"displayName"`
	Description         string     `json:"description"`
	ChannelID           string     `json:"channelId"`
	ChannelURI          string     `json:"channelUri"`
	EnginePath          string     `json:"enginePath"`
	ReleaseNotes        string     `json:"releaseNotes"`
	ThirdPartyNotices   string     `json:"thirdPartyNotices"`
	UpdateDate          time.Time  `json:"updateDate"`
	Catalog             Catalog    `json:"catalog"`
	Properties          Properties `json:"properties"`
}

// Catalog info from an installation.
type Catalog struct {
	BuildBranch                      string `json:"buildBranch"`
	BuildVersion                     string `json:"buildVersion"`
	ID                               string `json:"id"`
	LocalBuild                       string `json:"localBuild"`
	ManifestName                     string `json:"manifestName"`
	ManifestType                     string `json:"manifestType"`
	ProductDisplayVersion            string `json:"productDisplayVersion"`
	ProductLine                      string `json:"productLine"`
	ProductLineVersion               string `json:"productLineVersion"`
	ProductMilestone                 string `json:"productMilestone"`
	ProductMilestoneIsPrerelease     string `json:"productMilestoneIsPreRelease"`
	ProductName                      string `json:"productName"`
	ProductPatchVersion              string `json:"productPatchVersion"`
	ProductPreReleaseMilestoneSuffix string `json:"productPreReleaseMilestoneSuffix"`
	ProductSemanticVersion           string `json:"productSemanticVersion"`
	RequiredEngineVersion            string `json:"requiredEngineVersion"`
}

// Properties from an installation.
type Properties struct {
	CampaignID          string `json:"campaignId"`
	ChannelManifestID   string `json:"channelManifestId"`
	Nickname            string `json:"nickname"`
	SetupEngineFilePath string `json:"setupEngineFilePath"`
}

type searchOptions struct {
	all         bool
	prerelease  bool
	products    []string
	requires    []string
	requiresAny bool
	version     string
	latest      bool
	legacy      bool
}

// Option customizes the query to vswhere.
type Option func(so *searchOptions)

// WithAll finds all instances even if they are incomplete and may not launch.
func WithAll(all bool) Option {
	return func(so *searchOptions) { so.all = all }
}

// WithPrerelease also searches prerelease versions.
func WithPrerelease(prerelease bool) Option {
	return func(so *searchOptions) { so.prerelease = prerelease }
}

// WithProducts tries to find product IDs. A value of "*" by itself will instead
// search all product instances installed.
func WithProducts(products []string) Option {
	return func(so *searchOptions) { so.products = products }
}

// WithRequires enforces one or more workload/component IDs required when finding
// instances. By default, this requires AND matching all components unless
// WithRequiresAny is provided.
func WithRequires(requires []string) Option {
	return func(so *searchOptions) { so.requires = requires }
}

// WithRequiresAny will return an instance if they match any of the requirements
// provided with WithRequires.
func WithRequiresAny(requiresAny bool) Option {
	return func(so *searchOptions) { so.requiresAny = requiresAny }
}

// WithVersion specifies a version range for instances to find. For example,
// "[15.0,16.0)" will find all versions >=15.0 and <16.0.
func WithVersion(versionRange string) Option {
	return func(so *searchOptions) { so.version = versionRange }
}

// WithLatest only returns the newest version and the last one installed.
func WithLatest(latest bool) Option {
	return func(so *searchOptions) { so.latest = latest }
}

// WithLegacy will also search for Visual Studio 2015 and older products. Note
// that when doing this, return information is limited.
func WithLegacy(legacy bool) Option {
	return func(so *searchOptions) { so.legacy = legacy }
}

// Find finds all installations. Options can be provided to customize the search
// behavior.
func Find(ctx context.Context, options ...Option) ([]Installation, error) {
	var searchOpts searchOptions
	for _, o := range options {
		o(&searchOpts)
	}

	var args []string
	if searchOpts.all {
		args = append(args, "-all")
	}
	if searchOpts.prerelease {
		args = append(args, "-prerelease")
	}
	if len(searchOpts.products) > 0 {
		args = append(args, "-products")
		args = append(args, searchOpts.products...)
	}
	if len(searchOpts.requires) > 0 {
		args = append(args, "-requires")
		args = append(args, searchOpts.requires...)
	}
	if searchOpts.requiresAny {
		args = append(args, "-requiresAny")
	}
	if searchOpts.version != "" {
		args = append(args, "-version", searchOpts.version)
	}
	if searchOpts.latest {
		args = append(args, "-latest")
	}
	if searchOpts.legacy {
		args = append(args, "-legacy")
	}
	args = append(args, "-format", "json")
	return run(ctx, args)
}

// Get returns an indivdiual installation within a path. Returns an error if the
// installation wasn't found.
func Get(ctx context.Context, path string) (Installation, error) {
	installs, err := run(ctx, []string{"-path", path, "-format", "json"})
	if err != nil {
		return Installation{}, err
	}
	if len(installs) == 0 {
		return Installation{}, fmt.Errorf("no install at path %s", path)
	}
	return installs[0], nil
}

func run(ctx context.Context, args []string) ([]Installation, error) {
	vsWherePath := filepath.Join(
		os.Getenv("ProgramFiles(x86)"),
		"Microsoft Visual Studio",
		"Installer",
		"vswhere.exe",
	)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, vsWherePath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("vswhere failed: %s", string(stderr.Bytes()))
		}
		return nil, fmt.Errorf("vswhere failed: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))

	var installs []Installation
	if err := dec.Decode(&installs); err != nil {
		return nil, fmt.Errorf("failed parsing output of vswhere: %w", err)
	}
	return installs, nil
}
