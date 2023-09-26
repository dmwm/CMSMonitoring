package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"archive/tar" // Import the "tar" package for tar file operations
)


const baseUrl = "https://monit-grafana.cern.ch/api"

type Dashboard struct {
	Uid   string `json:"uid"`
	Title string `json:"title"`
}

func updatePathEnding(path string) string {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func getGrafanaAuth(fname string) (string, error) {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return "", fmt.Errorf("File %s does not exist", fname)
	}

	file, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var data map[string]interface{}
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return "", err
	}

	secretKey, ok := data["SECRET_KEY"].(string)
	if !ok {
		return "", fmt.Errorf("Failed to get SECRET_KEY from file")
	}

	headers := fmt.Sprintf("Bearer %s", secretKey)
	return headers, nil
}

func createBackUpFiles(dashboard Dashboard, folderTitle, headers string) {
	// Function to create backup files
}

func searchFoldersFromGrafana(headers string) {
	// Function to search folders and perform backup operations
}

func createTar(path, tarName string) error {
	with, err := os.Create(tarName)
	if err != nil {
		return err
	}
	defer with.Close()

	tw := tar.NewWriter(with)
	defer tw.Close()

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := addFileToTar(tw, filepath.Join(path, file.Name()), ""); err != nil {
			return err
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, path, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.Join(prefix, filepath.Base(path))

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}

func removeTempFiles(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	return nil
}

func getDate() (string, string, string) {
	gmt := time.Now().UTC()
	year := fmt.Sprintf("%04d", gmt.Year())
	mon := fmt.Sprintf("%02d", int(gmt.Month()))
	day := fmt.Sprintf("%02d", gmt.Day())
	return day, mon, year
}

func copyToHDFS(archive, hdfsPath string) error {
	day, mon, year := getDate()
	hdfsPath = updatePathEnding(hdfsPath)
	path := fmt.Sprintf("%s%s/%s/%s", hdfsPath, year, mon, day)

	if _, err := exec.Command("hadoop", "fs", "-mkdir", "-p", path).CombinedOutput(); err != nil {
		return err
	}

	copyCmd := exec.Command("hadoop", "fs", "-put", archive, path)
	copyCmd.Stdout = os.Stdout
	copyCmd.Stderr = os.Stderr

	if err := copyCmd.Run(); err != nil {
		return err
	}

	listCmd := exec.Command("hadoop", "fs", "-ls", path)
	listCmd.Stdout = os.Stdout
	listCmd.Stderr = os.Stderr
	if err := listCmd.Run(); err != nil {
		return err
	}

	return nil
}

func copyToFileSystem(archive, baseDir string) error {
	day, mon, year := getDate()
	baseDir = updatePathEnding(baseDir)
	path := fmt.Sprintf("%s%s/%s/%s", baseDir, year, mon, day)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	copyCmd := exec.Command("cp", archive, path)
	copyCmd.Stdout = os.Stdout
	copyCmd.Stderr = os.Stderr

	if err := copyCmd.Run(); err != nil {
		return err
	}

	listCmd := exec.Command("ls", path)
	listCmd.Stdout = os.Stdout
	listCmd.Stderr = os.Stderr
	if err := listCmd.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:  "Grafana Backup",
		Usage: "A tool to backup Grafana dashboards",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "token",
				Usage:    "Grafana token location JSON file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "filesystem-path",
				Usage:    "Path for Grafana backup in EOS",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			tokenFile := c.String("token")
			filesystemPath := c.String("filesystem-path")

			headers, err := getGrafanaAuth(tokenFile)
			if err != nil {
				return err
			}

			searchFoldersFromGrafana(headers)

			if err := createTar("./grafana", "grafanaBackup.tar.gz"); err != nil {
				return err
			}

			if err := copyToFileSystem("grafanaBackup.tar.gz", filesystemPath); err != nil {
				return err
			}

			if err := removeTempFiles("./grafana/"); err != nil {
				return err
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
