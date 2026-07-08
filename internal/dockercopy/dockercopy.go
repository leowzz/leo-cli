package dockercopy

import (
	"fmt"
	"strings"
)

const dockerHubRegistry = "docker.io"
const dockerHubLibraryPrefix = "library/"

type Copy struct {
	Source      string
	Destination string
}

func Resolve(registries map[string]string, source, destination string) (Copy, error) {
	sourceRef, err := parseSource(registries, source)
	if err != nil {
		return Copy{}, err
	}
	destinationRef, err := parseDestination(registries, destination, sourceRef)
	if err != nil {
		return Copy{}, err
	}

	return Copy{
		Source:      sourceRef.String(),
		Destination: destinationRef.String(),
	}, nil
}

type reference struct {
	registry   string
	repository string
	tag        string
	raw        string
}

func parseSource(registries map[string]string, value string) (reference, error) {
	if ref, ok := parseSourceAsAlias(registries, value); ok {
		return ref, nil
	}
	return parseSourceAsFullImage(value)
}

func parseSourceAsAlias(registries map[string]string, value string) (reference, bool) {
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return reference{}, false
	}

	registry, ok := registries[parts[0]]
	if !ok || registry == "" {
		return reference{}, false
	}

	repository, tag := splitRepositoryTag(parts[1])
	if repository == "" {
		return reference{}, false
	}

	return reference{registry: registry, repository: repository, tag: tag}, true
}

func parseSourceAsFullImage(value string) (reference, error) {
	if value == "" {
		return reference{}, fmt.Errorf("source image must not be empty")
	}

	ref, ok := splitFullImage(value)
	if !ok {
		return reference{}, fmt.Errorf("source image must be a valid image reference")
	}
	return ref, nil
}

func splitFullImage(value string) (reference, bool) {
	slash := strings.Index(value, "/")
	lastColon := strings.LastIndex(value, ":")

	if slash == -1 {
		if lastColon == -1 {
			return reference{}, false
		}
		repository := value[:lastColon]
		tag := value[lastColon+1:]
		if repository == "" || tag == "" {
			return reference{}, false
		}
		return reference{
			registry:   dockerHubRegistry,
			repository: repository,
			tag:        tag,
		}, true
	}

	firstPart := value[:slash]
	remainder := value[slash+1:]
	if firstPart == "" || remainder == "" {
		return reference{}, false
	}

	registry := ""
	repositoryAndTag := value
	if looksLikeRegistryDomain(firstPart) {
		registry = firstPart
		repositoryAndTag = remainder
	}

	repository := repositoryAndTag
	tag := ""
	if colon := strings.LastIndex(repositoryAndTag, ":"); colon != -1 && colon > strings.LastIndex(repositoryAndTag, "/") {
		repository = repositoryAndTag[:colon]
		tag = repositoryAndTag[colon+1:]
	}
	if repository == "" {
		return reference{}, false
	}

	return reference{registry: registry, repository: repository, tag: tag, raw: value}, true
}

func parseDestination(registries map[string]string, value string, source reference) (reference, error) {
	if value == "" {
		return reference{}, fmt.Errorf("destination image must be REGISTRY_OR_ALIAS[/REPOSITORY[:TAG]]")
	}

	parts := strings.SplitN(value, "/", 2)
	registry, err := resolveRegistry(registries, parts[0])
	if err != nil {
		return reference{}, err
	}

	destination := source
	destination.registry = registry
	destination.raw = ""
	if len(parts) == 2 {
		repository, tag := splitRepositoryTag(parts[1])
		if repository == "" {
			return reference{}, fmt.Errorf("destination image repository cannot be empty")
		}
		destination.repository = repository
		destination.tag = tag
	}

	return destination, nil
}

func resolveRegistry(registries map[string]string, value string) (string, error) {
	if registry, ok := registries[value]; ok {
		if registry == "" {
			return "", fmt.Errorf("registry alias %q is empty", value)
		}
		return registry, nil
	}
	if looksLikeRegistryDomain(value) {
		return value, nil
	}
	return "", fmt.Errorf("unknown registry alias %q; configure docker.registries.%s", value, value)
}

func looksLikeRegistryDomain(value string) bool {
	return strings.Contains(value, ".") || strings.Contains(value, ":") || value == "localhost"
}

func splitRepositoryTag(value string) (string, string) {
	slash := strings.LastIndex(value, "/")
	colon := strings.LastIndex(value, ":")
	if colon > slash {
		return value[:colon], value[colon+1:]
	}
	return value, ""
}

func (r reference) String() string {
	if r.raw != "" {
		return r.raw
	}
	repository := r.repository
	if r.registry == dockerHubRegistry && !strings.Contains(repository, "/") {
		repository = dockerHubLibraryPrefix + repository
	}
	if r.registry != "" {
		repository = r.registry + "/" + repository
	}
	if r.tag != "" {
		repository += ":" + r.tag
	}
	return repository
}
