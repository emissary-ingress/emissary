// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type DockerInspect struct {
	Id          string
	RepoTags    []string
	RepoDigests []string
}

func sort_u(in []string) []string {
	set := make(map[string]struct{}, len(in))
	for _, item := range in {
		set[item] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func Main() error {
	// 1. Get the "docker inspect" for all images
	bs, err := exec.Command("docker", "image", "ls", "--filter=dangling=false", "--format={{ .ID }}").Output()
	if err != nil {
		return err
	}
	ids := sort_u(strings.Split(strings.TrimSpace(string(bs)), "\n"))
	bs, err = exec.Command("docker", append([]string{"image", "inspect"}, ids...)...).Output()
	if err != nil {
		return err
	}
	var infos []DockerInspect
	if err := json.Unmarshal(bs, &infos); err != nil {
		return err
	}

	// 2. Decide what to do with each image
	workspacePull := make(map[string]struct{}) // pull these images from remote registries...
	workspaceTag := make(map[string]string)    // ... then tag them with these names
	workspaceLoad := make(map[string]struct{}) // store these images locally with 'docker image save'/'docker image load'

	for _, info := range infos {
		if len(info.RepoDigests) > 0 {
			workspacePull[info.RepoDigests[0]] = struct{}{}
			for _, tag := range info.RepoTags {
				workspaceTag[tag] = info.Id
			}
		} else {
			for _, tag := range info.RepoTags {
				workspaceLoad[tag] = struct{}{}
			}
		}
	}

	// 3. Record and do those things

	// Write the pull/tag steps to a file
	err = func() error {
		var lines []string
		for pull := range workspacePull {
			lines = append(lines, fmt.Sprintf("docker image pull %s\n", pull))
		}
		for tag, id := range workspaceTag {
			lines = append(lines, fmt.Sprintf("docker image tag %s %s\n", id, tag))
		}
		sort.Strings(lines) // NB: relying on "pull" sorting before "tag"

		lines = append([]string{
			"#!/usr/bin/env bash\n",
			"set -ex\n",
		}, lines...)

		restoreSh, err := os.OpenFile("docker/images.sh", os.O_CREATE|os.O_WRONLY, 0777)
		if err != nil {
			return err
		}
		defer restoreSh.Close()
		for _, line := range lines {
			if _, err := io.WriteString(restoreSh, line); err != nil {
				return err
			}
		}

		return nil
	}()
	if err != nil {
		return err
	}

	// Run 'docker image save'
	err = func() error {
		localImages := make([]string, 0, len(workspaceLoad))
		for image := range workspaceLoad {
			localImages = append(localImages, image)
		}
		sort.Strings(localImages)

		restoreTar, err := os.OpenFile("docker/images.tar", os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer restoreTar.Close()

		cmd := exec.Command("docker", append([]string{"image", "save"}, localImages...)...)
		cmd.Stdout = restoreTar
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}
