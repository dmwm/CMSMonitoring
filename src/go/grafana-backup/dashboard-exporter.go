package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"archive/tar" // Import the "tar" package for tar file operations

	"github.com/urfave/cli/v2"
)

const (
	baseUrl       = "https://monit-grafana.cern.ch/api"
	grafanaFolder = "./grafana"
	tarFileName   = "grafanaBackup.tar.gz"
)

type Dashboard struct {
	Uid   string `json:"uid"`
	Title string `json:"title"`
}

func getGrafanaAuth(fname string) (string, error) {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return "", fmt.Errorf("file %s does not exist", fname)
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
		return "", fmt.Errorf("failed to get SECRET_KEY from file")
	}

	headers := fmt.Sprintf("Bearer %s", secretKey)
	return headers, nil
}

func createBackUpFiles(dashboard Dashboard, folderTitle, headers string) error {
	allJson := []Dashboard{}

	dashboardUid := dashboard.Uid
	path := filepath.Join("grafana", "dashboards", folderTitle)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	dashboardUrl := filepath.Join(baseUrl, "dashboards/uid", dashboardUid)
	fmt.Println("dashboardUrl", dashboardUrl)
	req, err := http.NewRequest("GET", dashboardUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", headers)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var dashboardData Dashboard
	if err := json.Unmarshal(body, &dashboardData); err != nil {
		return err
	}

	titleOfDashboard := strings.ReplaceAll(dashboardData.Title, " ", "_")
	titleOfDashboard = strings.ReplaceAll(titleOfDashboard, ".", "_")
	titleOfDashboard = strings.ReplaceAll(titleOfDashboard, "/", "_")
	uidOfDashboard := dashboardData.Uid

	filenameForFolder := fmt.Sprintf("%s/%s-%s.json", path, titleOfDashboard, uidOfDashboard)
	fmt.Println("filenameForFolder", filenameForFolder)
	if err := os.WriteFile(filenameForFolder, body, os.ModePerm); err != nil {
		return err
	}

	fmt.Println(strings.Repeat("*", 10) + "\n")
	allJson = append(allJson, dashboardData)

	filenameForAllJson := filepath.Join(path, "all.json")
	fmt.Println("filenameForAllJson", filenameForAllJson)
	allJsonBytes, err := json.Marshal(allJson)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filenameForAllJson, allJsonBytes, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func searchFoldersFromGrafana(headers string) error {
	foldersUrl := filepath.Join(baseUrl, "search?folderIds=0&orgId=11")

	req, err := http.NewRequest("GET", foldersUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", headers)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request %s, failed with reason: %s", foldersUrl, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var foldersJson []Dashboard
	if err := json.Unmarshal(body, &foldersJson); err != nil {
		return err
	}

	for _, folder := range foldersJson {
		folderId := folder.Uid
		folderTitle := folder.Title

		if folderTitle != "Production" && folderTitle != "Development" && folderTitle != "Playground" && folderTitle != "Backup" {
			folderTitle = "General"
			dashboard := folder
			if err := createBackUpFiles(dashboard, folderTitle, headers); err != nil {
				return err
			}
		} else {
			dashboardQueryUrl := filepath.Join(baseUrl, "search?folderIds="+folderId+"&orgId=11&query=")

			fmt.Println("individualFolderUrl", dashboardQueryUrl)

			req, err := http.NewRequest("GET", dashboardQueryUrl, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", headers)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("request %s, failed with reason: %s", dashboardQueryUrl, resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			pathToDb := filepath.Join("grafana")
			if err := os.MkdirAll(pathToDb, os.ModePerm); err != nil {
				return err
			}

			filenameForFolder := filepath.Join(pathToDb, folderId+"-"+folderTitle+".json")
			fmt.Println("filenameForFolder", filenameForFolder)
			if err := os.WriteFile(filenameForFolder, body, os.ModePerm); err != nil {
				return err
			}

			var folderData []Dashboard
			if err := json.Unmarshal(body, &folderData); err != nil {
				return err
			}

			dashboardsAmount := len(folderData)
			for index, dashboard := range folderData {
				folderTitle := dashboard.Title
				fmt.Printf("Index/Amount: %d/%d\n", index, dashboardsAmount)

				if err := createBackUpFiles(dashboard, folderTitle, headers); err != nil {
					return err
				}
			}
		}
	}

	return nil
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

func copyToFileSystem(archive, baseDir string) error {
	day, mon, year := getDate()
	path := filepath.Join(baseDir, year, mon, day)

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

func backupGrafana(headers, filesystemPath string) error {
	searchFoldersFromGrafana(headers)

	if err := createTar(grafanaFolder, tarFileName); err != nil {
		return fmt.Errorf("error creating tar: %v", err)
	}

	if err := copyToFileSystem(tarFileName, filesystemPath); err != nil {
		return fmt.Errorf("error copying to filesystem: %v", err)
	}

	if err := removeTempFiles(grafanaFolder); err != nil {
		return fmt.Errorf("error removing temp files: %v", err)
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
				log.Fatalf("Error getting Grafana auth: %v", err)
			}

			if err := backupGrafana(headers, filesystemPath); err != nil {
				log.Fatalf("Error backing up Grafana: %v", err)
			}

			fmt.Println("Backup completed successfully")
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
