package blocker

/*
 * allowlist.go — System processes that must NEVER be killed.
 *
 * CONCEPT: When silo is sealed, it kills any process not in the
 * workspace's allowed_apps list. But some processes are the OS itself —
 * killing Finder crashes macOS, killing explorer.exe bricks Windows.
 *
 * This allowlist is intentionally GENEROUS. It's better to miss
 * blocking one obscure app than to crash the user's OS.
 *
 * The allowlist works by checking if a running process name matches
 * any entry. We check with case-insensitive prefix/suffix matching
 * because process names vary across versions.
 */

import "strings"

// systemAllowlistDarwin contains macOS processes that must never be killed.
var systemAllowlistDarwin = []string{
	// Core OS
	"kernel_task",
	"launchd",
	"loginwindow",
	"WindowServer",
	"SystemUIServer",
	"Finder",
	"Dock",
	"Spotlight",
	"mds",
	"mds_stores",
	"mdworker",
	"notifyd",
	"cfprefsd",
	"coreservicesd",
	"opendirectoryd",
	"diskarbitrationd",
	"fseventsd",
	"powerd",
	"thermald",
	"bluetoothd",
	"bluetoothaudiod",
	"airportd",
	"WiFiAgent",
	"UserEventAgent",
	"universalaccessd",
	"AXVisualSupportAgent",
	"coreauthd",
	"securityd",
	"trustd",
	"secd",
	"lsd",
	"iconservicesagent",
	"distnoted",
	"usernoted",
	"nsurlsessiond",
	"nsurlstoraged",
	"CommCenter",
	"sharingd",
	"rapportd",
	"IMDPersistenceAgent",
	"pboard",
	"corebrightnessd",
	"TouchBarServer",
	"ControlCenter",
	"sysmond",
	"sandboxd",
	"taskgated",

	// Display & Graphics
	"coreaudiod",
	"audiomxd",
	"mediaremoted",
	"AMPDevicesAgent",

	// Input
	"hidd",

	// Networking
	"mDNSResponder",
	"configd",
	"networkd",
	"symptomsd",

	// Wails / silo itself
	"silo",
	"WebKit",
	"webkit",

	// Helper processes that may be needed
	"softwareupdated",
	"appstored",
	"installd",
	"pkd",
	"syslogd",
	"logd",
	"diagnosticd",
	"spindump",
	"ReportCrash",
}

// systemAllowlistWindows contains Windows processes that must never be killed.
var systemAllowlistWindows = []string{
	"explorer.exe",
	"csrss.exe",
	"winlogon.exe",
	"dwm.exe",
	"svchost.exe",
	"services.exe",
	"lsass.exe",
	"smss.exe",
	"wininit.exe",
	"RuntimeBroker.exe",
	"ShellExperienceHost.exe",
	"StartMenuExperienceHost.exe",
	"SearchHost.exe",
	"SearchIndexer.exe",
	"taskhostw.exe",
	"ctfmon.exe",
	"conhost.exe",
	"fontdrvhost.exe",
	"sihost.exe",
	"dllhost.exe",
	"WmiPrvSE.exe",
	"spoolsv.exe",
	"System",
	"System Idle Process",
	"Registry",
	"silo.exe",
	"WebView2",
}

// IsSystemProcess checks if a process name is in the system allowlist.
// Uses case-insensitive matching and checks for substring matches
// (because macOS reports "com.apple.Finder" not just "Finder").
func IsSystemProcess(name string) bool {
	lower := strings.ToLower(name)

	// Check against platform-appropriate list
	for _, sys := range systemAllowlistDarwin {
		if strings.Contains(lower, strings.ToLower(sys)) {
			return true
		}
	}
	for _, sys := range systemAllowlistWindows {
		if strings.Contains(lower, strings.ToLower(sys)) {
			return true
		}
	}
	return false
}

// IsAllowedApp checks if a process matches one of the workspace's allowed apps.
// Uses fuzzy matching: "Google Chrome" matches "Google Chrome Helper".
func IsAllowedApp(processName string, allowedApps []string) bool {
	lower := strings.ToLower(processName)
	for _, app := range allowedApps {
		appLower := strings.ToLower(app)
		if strings.Contains(lower, appLower) || strings.Contains(appLower, lower) {
			return true
		}
	}
	return false
}
