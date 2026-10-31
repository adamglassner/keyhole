; Keyhole installer (NSIS 3, Modern UI 2)
; Build:  makensis -DVERSION=1.0.0 -DSRCDIR=..\dist\Keyhole -DOUTDIR=..\dist installer\keyhole.nsi
; Produces Keyhole-Setup-<version>.exe. The Ironite Launcher runs it silently with /S /D=<folder>.

Unicode true
ManifestDPIAware true
SetCompressor /SOLID lzma
RequestExecutionLevel user

!ifndef VERSION
  !define VERSION "1.0.0"
!endif
!ifndef SRCDIR
  !define SRCDIR "..\dist\Keyhole"
!endif
!ifndef OUTDIR
  !define OUTDIR "."
!endif

!define APPNAME "Keyhole"
!define PUBLISHER "Ironite"
!define WEBSITE "https://github.com/adamglassner/keyhole"
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keyhole"

Name "${APPNAME}"
OutFile "${OUTDIR}\Keyhole-Setup-${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Programs\Keyhole"
InstallDirRegKey HKCU "Software\Ironite\Keyhole" "InstallDir"
BrandingText "Keyhole ${VERSION}  -  The Daily Word Vault"
ShowInstDetails nevershow
ShowUninstDetails nevershow

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APPNAME}"
VIAddVersionKey "CompanyName" "${PUBLISHER}"
VIAddVersionKey "FileDescription" "Keyhole Setup"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "(c) 2026 ${PUBLISHER}"

!include "MUI2.nsh"
!include "FileFunc.nsh"

; ---------------------------------------------------------------- theme
!define MUI_ICON "..\branding\keyhole.ico"
!define MUI_UNICON "..\branding\keyhole.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "..\branding\installer_header.bmp"
!define MUI_HEADERIMAGE_UNBITMAP "..\branding\installer_header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "..\branding\installer_welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "..\branding\installer_welcome.bmp"
!define MUI_INSTFILESPAGE_PROGRESSBAR "smooth"
!define MUI_ABORTWARNING
!define MUI_ABORTWARNING_TEXT "Are you sure you want to stop installing Keyhole?"

; ---------------------------------------------------------------- pages
!define MUI_WELCOMEPAGE_TITLE "Welcome to Keyhole!"
!define MUI_WELCOMEPAGE_TEXT "Keyhole is a daily word game. Crack a new vault every day: three locks, a 4-, 5- and 6-letter word, and 15 guesses to share between them.$\r$\n$\r$\n\
Open the vault with guesses to spare to earn gold keys and unlock new lock styles. Practice vaults are unlimited.$\r$\n$\r$\n\
Click Next to continue."
!insertmacro MUI_PAGE_WELCOME

!define MUI_DIRECTORYPAGE_TEXT_TOP "Keyhole will be installed for your Windows account in the folder below. No administrator rights are needed."
!insertmacro MUI_PAGE_DIRECTORY

!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_TITLE "Keyhole is ready!"
!define MUI_FINISHPAGE_TEXT "Today's vault is waiting. Your stats, streak and gold keys are kept safe between updates."
!define MUI_FINISHPAGE_RUN "$INSTDIR\Keyhole.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Start Keyhole now"
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Create a desktop shortcut"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION CreateDesktopShortcut
!insertmacro MUI_PAGE_FINISH

!define MUI_WELCOMEPAGE_TITLE "Uninstall Keyhole"
!define MUI_WELCOMEPAGE_TEXT "This will remove Keyhole from your computer.$\r$\n$\r$\nClick Next to continue."
!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

Function CreateDesktopShortcut
  CreateShortcut "$DESKTOP\Keyhole.lnk" "$INSTDIR\Keyhole.exe" "" "$INSTDIR\Keyhole.exe" 0
FunctionEnd

; ---------------------------------------------------------------- install
Section "Keyhole" SecMain
  SectionIn RO
  ; close a running copy (for example when updating from inside the game)
  nsExec::Exec 'taskkill /IM Keyhole.exe /F'
  Pop $0
  Sleep 300

  SetOutPath "$INSTDIR"
  File "${SRCDIR}\Keyhole.exe"
  File /nonfatal "${SRCDIR}\LICENSE.txt"
  File /nonfatal "${SRCDIR}\CHANGELOG.md"
  File /nonfatal "${SRCDIR}\THIRD-PARTY-NOTICES.txt"

  WriteUninstaller "$INSTDIR\Uninstall Keyhole.exe"
  IfSilent +2
  CreateShortcut "$SMPROGRAMS\Keyhole.lnk" "$INSTDIR\Keyhole.exe" "" "$INSTDIR\Keyhole.exe" 0

  WriteRegStr HKCU "Software\Ironite\Keyhole" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayName" "Keyhole"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINSTKEY}" "Publisher" "${PUBLISHER}"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayIcon" "$INSTDIR\Keyhole.exe"
  WriteRegStr HKCU "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTKEY}" "URLInfoAbout" "${WEBSITE}"
  WriteRegStr HKCU "${UNINSTKEY}" "UninstallString" '"$INSTDIR\Uninstall Keyhole.exe"'
  WriteRegStr HKCU "${UNINSTKEY}" "QuietUninstallString" '"$INSTDIR\Uninstall Keyhole.exe" /S'
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "${UNINSTKEY}" "EstimatedSize" "$0"
SectionEnd

; ---------------------------------------------------------------- uninstall
Section "Uninstall"
  nsExec::Exec 'taskkill /IM Keyhole.exe /F'
  Pop $0
  Delete "$SMPROGRAMS\Keyhole.lnk"
  Delete "$DESKTOP\Keyhole.lnk"
  Delete "$INSTDIR\Keyhole.exe"
  Delete "$INSTDIR\LICENSE.txt"
  Delete "$INSTDIR\CHANGELOG.md"
  Delete "$INSTDIR\THIRD-PARTY-NOTICES.txt"
  Delete "$INSTDIR\Uninstall Keyhole.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${UNINSTKEY}"
  DeleteRegKey HKCU "Software\Ironite\Keyhole"

  IfSilent done
  MessageBox MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2 "Also delete your stats, streak and gold keys?$\r$\n$\r$\nChoose No to keep them in case you reinstall Keyhole later." IDNO done
  RMDir /r "$APPDATA\Ironite\Keyhole"
  done:
SectionEnd
