package scripts_test

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMenuHelpListsTopLevelSections(t *testing.T) {
	cmd := exec.Command("bash", "menu.sh", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("menu help returned error: %v, output: %s", err, output)
	}

	text := string(output)
	for _, want := range []string{
		"快捷安装 / 修复",
		"核心管理",
		"域名与证书",
		"协议与端口设置",
		"用户与订阅管理",
		"流量与维护",
		"生成 sing-box 配置",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected menu help to contain %q, got %s", want, text)
		}
	}
}

func TestInstallHelpListsLifecycleActions(t *testing.T) {
	cmd := exec.Command("bash", "install.sh", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install help returned error: %v, output: %s", err, output)
	}

	text := string(output)
	for _, want := range []string{
		"安装",
		"重装 sing-box",
		"更新 sing-box",
		"卸载 sing-box",
		"生成配置",
		"刷新流量",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected install help to contain %q, got %s", want, text)
		}
	}
}

func TestSb2subHelpListsCommandGroups(t *testing.T) {
	cmd := exec.Command("bash", "sb2sub", "--help")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sb2sub help returned error: %v, output: %s", err, output)
	}

	text := string(output)
	for _, want := range []string{
		"service",
		"user",
		"sub",
		"traffic",
		"server",
		"menu",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected sb2sub help to contain %q, got %s", want, text)
		}
	}
}

func TestInstallValidateUsesBundledBinary(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, "#!/usr/bin/env bash\nprintf 'fake version\\n'")

	cmd := exec.Command("/bin/bash", "install.sh", "validate")
	cmd.Dir = filepath.Join(repoDir, "scripts")
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install validate returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "环境检查通过") {
		t.Fatalf("expected validate output to confirm success, got %s", output)
	}
}

func TestGenerateConfigUsesBundledBinary(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, "#!/usr/bin/env bash\nprintf '{\"version\":\"bundle\"}\\n'")

	baseDir := filepath.Join(t.TempDir(), "runtime")
	cmd := exec.Command("/bin/bash", "install.sh", "generate-config")
	cmd.Dir = filepath.Join(repoDir, "scripts")
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin", "SB2SUB_BASE_DIR="+baseDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate-config returned error: %v, output: %s", err, output)
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "etc", "sing-box.json"))
	if err != nil {
		t.Fatalf("read generated sing-box config: %v", err)
	}
	if string(data) != "{\"version\":\"bundle\"}\n" {
		t.Fatalf("expected bundled binary output in sing-box config, got %q", string(data))
	}
}

func TestBuildReleaseBundle(t *testing.T) {
	tmpDir := t.TempDir()
	version := "v0.0.0-test"
	archiveFile := filepath.Join(tmpDir, "dist", "sb2sub_"+version+"_linux_"+runtime.GOARCH+".tar.gz")

	cmd := exec.Command("/bin/bash", "build-release.sh", version)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(),
		"OUTPUT_DIR="+filepath.Join(tmpDir, "dist"),
		"TARGETS=linux/"+runtime.GOARCH,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build-release returned error: %v, output: %s", err, output)
	}

	entries := releaseArchiveEntries(t, archiveFile)
	for _, want := range []string{
		"sb2sub/README.md",
		"sb2sub/bin/sb2subd",
		"sb2sub/scripts/get-latest.sh",
		"sb2sub/scripts/install.sh",
		"sb2sub/scripts/menu.sh",
		"sb2sub/scripts/sb2sub",
		"sb2sub/packaging/systemd/sb2subd.service",
	} {
		if !entries[want] {
			t.Fatalf("expected release archive to contain %q, got %v", want, entries)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "dist", "checksums.txt")); err != nil {
		t.Fatalf("stat checksums: %v", err)
	}
}

func TestInstallCreatesGlobalCommandAndService(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, fakeDaemonScript())
	if err := os.Chmod(filepath.Join(repoDir, "scripts", "sb2sub"), 0o644); err != nil {
		t.Fatalf("chmod copied sb2sub script: %v", err)
	}

	baseDir := filepath.Join(t.TempDir(), "runtime")
	systemdDir := filepath.Join(t.TempDir(), "systemd")
	binLinkDir := filepath.Join(t.TempDir(), "bin-link")
	toolsDir := filepath.Join(t.TempDir(), "tools")
	systemctlLog := filepath.Join(t.TempDir(), "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	cmd := exec.Command("/bin/bash", "install.sh", "install")
	cmd.Dir = filepath.Join(repoDir, "scripts")
	cmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install returned error: %v, output: %s", err, output)
	}

	if _, err := os.Lstat(filepath.Join(binLinkDir, "sb2sub")); err != nil {
		t.Fatalf("stat global sb2sub command: %v", err)
	}

	serviceFile := filepath.Join(systemdDir, "sb2subd.service")
	serviceData, err := os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}
	serviceText := string(serviceData)
	for _, want := range []string{
		filepath.Join(repoDir, "bin", "sb2subd"),
		"--base-dir " + baseDir,
	} {
		if !strings.Contains(serviceText, want) {
			t.Fatalf("expected service file to contain %q, got %s", want, serviceText)
		}
	}

	logData, err := os.ReadFile(systemctlLog)
	if err != nil {
		t.Fatalf("read systemctl log: %v", err)
	}
	for _, want := range []string{
		"daemon-reload",
		"enable sb2subd.service",
		"restart sb2subd.service",
	} {
		if !strings.Contains(string(logData), want) {
			t.Fatalf("expected systemctl log to contain %q, got %s", want, logData)
		}
	}

	statusCmd := exec.Command(filepath.Join(binLinkDir, "sb2sub"), "status")
	statusCmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sb2sub status returned error: %v, output: %s", err, statusOutput)
	}
	if !strings.Contains(string(statusOutput), "fake systemctl status") {
		t.Fatalf("expected sb2sub status to return stub output, got %s", statusOutput)
	}
}

func TestGetLatestInstallerPullsReleaseAndInstalls(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")
	downloadDir := filepath.Join(tmpDir, "downloads")
	version := "v0.0.0-test"
	releaseJSON := filepath.Join(tmpDir, "latest.json")
	installDir := filepath.Join(tmpDir, "install")
	baseDir := filepath.Join(tmpDir, "runtime")
	systemdDir := filepath.Join(tmpDir, "systemd")
	binLinkDir := filepath.Join(tmpDir, "bin-link")
	toolsDir := filepath.Join(tmpDir, "tools")
	systemctlLog := filepath.Join(tmpDir, "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	buildCmd := exec.Command("/bin/bash", "build-release.sh", version)
	buildCmd.Dir = "."
	buildCmd.Env = append(os.Environ(),
		"OUTPUT_DIR="+distDir,
		"TARGETS=linux/"+runtime.GOARCH,
	)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build test release returned error: %v, output: %s", err, buildOutput)
	}
	if err := os.MkdirAll(filepath.Join(downloadDir, version), 0o755); err != nil {
		t.Fatalf("create fake download dir: %v", err)
	}
	archiveName := "sb2sub_" + version + "_linux_" + runtime.GOARCH + ".tar.gz"
	copyArchiveCmd := exec.Command("cp", filepath.Join(distDir, archiveName), filepath.Join(downloadDir, version, archiveName))
	if output, err := copyArchiveCmd.CombinedOutput(); err != nil {
		t.Fatalf("copy release archive: %v, output: %s", err, output)
	}

	if err := os.WriteFile(releaseJSON, []byte("{\"tag_name\":\""+version+"\"}\n"), 0o644); err != nil {
		t.Fatalf("write release json: %v", err)
	}

	cmd := exec.Command("/bin/bash", "get-latest.sh")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
		"SB2SUB_RELEASE_API=file://"+releaseJSON,
		"SB2SUB_RELEASE_DOWNLOAD_BASE=file://"+downloadDir,
		"SB2SUB_INSTALL_DIR="+installDir,
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get-latest returned error: %v, output: %s", err, output)
	}

	if _, err := os.Stat(filepath.Join(installDir, "bin", "sb2subd")); err != nil {
		t.Fatalf("stat installed daemon: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(binLinkDir, "sb2sub")); err != nil {
		t.Fatalf("stat installed global command: %v", err)
	}
}

func TestSb2subCommandManagementFlow(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, fakeDaemonScript())

	baseDir := filepath.Join(t.TempDir(), "runtime")
	systemdDir := filepath.Join(t.TempDir(), "systemd")
	binLinkDir := filepath.Join(t.TempDir(), "bin-link")
	toolsDir := filepath.Join(t.TempDir(), "tools")
	systemctlLog := filepath.Join(t.TempDir(), "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	installCmd := exec.Command("/bin/bash", "install.sh", "install")
	installCmd.Dir = filepath.Join(repoDir, "scripts")
	installCmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("install returned error: %v, output: %s", err, output)
	}

	sb2sub := filepath.Join(binLinkDir, "sb2sub")
	commonEnv := append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
		"SB2SUB_AUTO_CONFIRM=1",
	)

	addUserCmd := exec.Command(sb2sub, "user", "add", "--name", "alice", "--note", "phone", "--quota", "10G", "--days", "30")
	addUserCmd.Env = commonEnv
	output, err := addUserCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("user add returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "已创建用户") {
		t.Fatalf("expected user add output to contain success message, got %s", output)
	}

	userListCmd := exec.Command(sb2sub, "user", "list")
	userListCmd.Env = commonEnv
	output, err = userListCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("user list returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "alice") {
		t.Fatalf("expected user list output to contain alice, got %s", output)
	}

	addSubCmd := exec.Command(sb2sub, "sub", "add", "--user", "alice", "--type", "clash", "--name", "alice-clash")
	addSubCmd.Env = commonEnv
	output, err = addSubCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sub add returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "订阅地址") {
		t.Fatalf("expected sub add output to contain full URL, got %s", output)
	}

	subListCmd := exec.Command(sb2sub, "sub", "list")
	subListCmd.Env = commonEnv
	output, err = subListCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sub list returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "alice-clash") {
		t.Fatalf("expected sub list output to contain subscription name, got %s", output)
	}

	trafficCmd := exec.Command(sb2sub, "traffic", "show")
	trafficCmd.Env = commonEnv
	output, err = trafficCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("traffic show returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "总量") {
		t.Fatalf("expected traffic show output to contain total column, got %s", output)
	}

	serverDomainCmd := exec.Command(sb2sub, "server", "domain", "--value", "example.com")
	serverDomainCmd.Env = commonEnv
	output, err = serverDomainCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("server domain returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "已更新域名") {
		t.Fatalf("expected server domain output to contain success message, got %s", output)
	}

	serverShowCmd := exec.Command(sb2sub, "server", "show")
	serverShowCmd.Env = commonEnv
	output, err = serverShowCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("server show returned error: %v, output: %s", err, output)
	}
	for _, want := range []string{"example.com", "VLESS", "Hysteria2"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("expected server show output to contain %q, got %s", want, output)
		}
	}

	dbFile := filepath.Join(baseDir, "var", "sb2sub.db")
	row := openTestDB(t, dbFile).QueryRow(`select count(*) from users where username = 'alice'`)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan user count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one alice user, got %d", count)
	}

	configData, err := os.ReadFile(filepath.Join(baseDir, "etc", "config.yaml"))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(configData), "domain: example.com") {
		t.Fatalf("expected config file to contain updated domain, got %s", configData)
	}
}

func TestSb2subPromptAddsUserWhenFlagsMissing(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, fakeDaemonScript())

	baseDir := filepath.Join(t.TempDir(), "runtime")
	systemdDir := filepath.Join(t.TempDir(), "systemd")
	binLinkDir := filepath.Join(t.TempDir(), "bin-link")
	toolsDir := filepath.Join(t.TempDir(), "tools")
	systemctlLog := filepath.Join(t.TempDir(), "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	installCmd := exec.Command("/bin/bash", "install.sh", "install")
	installCmd.Dir = filepath.Join(repoDir, "scripts")
	installCmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("install returned error: %v, output: %s", err, output)
	}

	cmd := exec.Command(filepath.Join(binLinkDir, "sb2sub"), "user", "add")
	cmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
		"SB2SUB_AUTO_CONFIRM=1",
	)
	cmd.Stdin = strings.NewReader("bob\nsecond\n15G\n45\ny\ny\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("interactive user add returned error: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "已创建用户") {
		t.Fatalf("expected interactive user add output to contain success, got %s", output)
	}

	row := openTestDB(t, filepath.Join(baseDir, "var", "sb2sub.db")).QueryRow(`select count(*) from users where username = 'bob'`)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan prompt user count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected prompt-created bob user, got %d", count)
	}
}

func TestSb2subServerProtocolAndPortCommandsPersistConfig(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, fakeDaemonScript())

	baseDir := filepath.Join(t.TempDir(), "runtime")
	systemdDir := filepath.Join(t.TempDir(), "systemd")
	binLinkDir := filepath.Join(t.TempDir(), "bin-link")
	toolsDir := filepath.Join(t.TempDir(), "tools")
	systemctlLog := filepath.Join(t.TempDir(), "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	installCmd := exec.Command("/bin/bash", "install.sh", "install")
	installCmd.Dir = filepath.Join(repoDir, "scripts")
	installCmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("install returned error: %v, output: %s", err, output)
	}

	sb2sub := filepath.Join(binLinkDir, "sb2sub")
	commonEnv := append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
		"SB2SUB_AUTO_CONFIRM=1",
	)

	disableVLESSCmd := exec.Command(sb2sub, "server", "protocol", "--name", "vless", "--enabled", "false")
	disableVLESSCmd.Env = commonEnv
	if output, err := disableVLESSCmd.CombinedOutput(); err != nil {
		t.Fatalf("server protocol returned error: %v, output: %s", err, output)
	}

	setPortCmd := exec.Command(sb2sub, "server", "port", "--name", "hysteria2", "--value", "9443")
	setPortCmd.Env = commonEnv
	if output, err := setPortCmd.CombinedOutput(); err != nil {
		t.Fatalf("server port returned error: %v, output: %s", err, output)
	}

	showCmd := exec.Command(sb2sub, "server", "show")
	showCmd.Env = commonEnv
	output, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("server show returned error: %v, output: %s", err, output)
	}
	text := string(output)
	for _, want := range []string{"VLESS: 关闭", "Hysteria2: 开启 (端口 9443)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected server show output to contain %q, got %s", want, output)
		}
	}

	configData, err := os.ReadFile(filepath.Join(baseDir, "etc", "config.yaml"))
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	for _, want := range []string{"enabled: false", "listen_port: 9443"} {
		if !strings.Contains(string(configData), want) {
			t.Fatalf("expected config file to contain %q, got %s", want, configData)
		}
	}
}

func TestSb2subMenuOpensAndCanExit(t *testing.T) {
	repoDir := prepareScriptRepo(t)
	writeFakeDaemon(t, repoDir, fakeDaemonScript())

	baseDir := filepath.Join(t.TempDir(), "runtime")
	systemdDir := filepath.Join(t.TempDir(), "systemd")
	binLinkDir := filepath.Join(t.TempDir(), "bin-link")
	toolsDir := filepath.Join(t.TempDir(), "tools")
	systemctlLog := filepath.Join(t.TempDir(), "systemctl.log")
	writeFakeSystemctl(t, toolsDir, systemctlLog)

	installCmd := exec.Command("/bin/bash", "install.sh", "install")
	installCmd.Dir = filepath.Join(repoDir, "scripts")
	installCmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("install returned error: %v, output: %s", err, output)
	}

	cmd := exec.Command(filepath.Join(binLinkDir, "sb2sub"), "menu")
	cmd.Env = append(os.Environ(),
		"PATH="+toolsDir+":/usr/bin:/bin",
		"SB2SUB_BASE_DIR="+baseDir,
		"SB2SUB_SYSTEMD_DIR="+systemdDir,
		"SB2SUB_BIN_LINK_DIR="+binLinkDir,
		"FAKE_SYSTEMCTL_LOG="+systemctlLog,
	)
	cmd.Stdin = strings.NewReader("0\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("menu returned error: %v, output: %s", err, output)
	}

	text := string(output)
	for _, want := range []string{"快捷安装 / 修复", "用户与订阅管理", "流量与维护"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected menu output to contain %q, got %s", want, output)
		}
	}
}

func prepareScriptRepo(t *testing.T) string {
	t.Helper()

	repoDir := filepath.Join(t.TempDir(), "repo")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, "bin"), 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	copyCmd := exec.Command("cp", "-R", cwd, filepath.Join(repoDir, "scripts"))
	if output, err := copyCmd.CombinedOutput(); err != nil {
		t.Fatalf("copy scripts directory: %v, output: %s", err, output)
	}
	packagingDir := filepath.Join(filepath.Dir(cwd), "packaging")
	copyPackagingCmd := exec.Command("cp", "-R", packagingDir, filepath.Join(repoDir, "packaging"))
	if output, err := copyPackagingCmd.CombinedOutput(); err != nil {
		t.Fatalf("copy packaging directory: %v, output: %s", err, output)
	}
	return repoDir
}

func writeFakeDaemon(t *testing.T, repoDir string, content string) {
	t.Helper()

	binFile := filepath.Join(repoDir, "bin", "sb2subd")
	if err := os.WriteFile(binFile, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake daemon: %v", err)
	}
}

func fakeDaemonScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "--version" ]]; then
  printf 'sb2subd fake\n'
  exit 0
fi
if [[ "${1:-}" == "--mode" && "${2:-}" == "render-singbox" ]]; then
  printf '{"version":"bundle"}\n'
  exit 0
fi
printf 'fake daemon\n'
`
}

func writeFakeSystemctl(t *testing.T, toolsDir string, logFile string) {
	t.Helper()

	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatalf("create tools dir: %v", err)
	}
	content := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${FAKE_SYSTEMCTL_LOG}"
if [[ "${1:-}" == "status" ]]; then
  printf 'fake systemctl status\n'
fi
`
	if err := os.WriteFile(filepath.Join(toolsDir, "systemctl"), []byte(content), 0o755); err != nil {
		t.Fatalf("write fake systemctl: %v", err)
	}
	if err := os.WriteFile(logFile, nil, 0o644); err != nil {
		t.Fatalf("init fake systemctl log: %v", err)
	}
}

func releaseArchiveEntries(t *testing.T, archiveFile string) map[string]bool {
	t.Helper()

	file, err := os.Open(archiveFile)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	entries := map[string]bool{}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return entries
			}
			t.Fatalf("read tar entry: %v", err)
		}
		entries[header.Name] = true
	}
}

func openTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}
