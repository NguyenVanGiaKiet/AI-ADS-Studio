#pragma codepage 65001
#define StageDir "{{STAGE_DIR}}"
#define OutputDir "{{OUTPUT_DIR}}"
#define GroqModel "{{GROQ_MODEL}}"

[Setup]
AppId={{D412D26A-AC3D-4D22-A06A-25B44B46E0A1}
AppName=AI ADS Studio
AppVersion=1.0.0
AppPublisher=AI ADS Studio
DefaultDirName={localappdata}\Programs\AI ADS Studio
DefaultGroupName=AI ADS Studio
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir={#OutputDir}
OutputBaseFilename=AI-ADS-Studio-Setup-Windows
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\ai-ads-studio.ico
SetupIconFile={#StageDir}\ai-ads-studio.ico
CloseApplications=yes
RestartApplications=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Dirs]
Name: "{app}\storage\outputs"; Flags: uninsneveruninstall

[Files]
Source: "{#StageDir}\AI-ADS-Studio.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\Start.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\README.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\ai-ads-studio.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\groq-key.txt"; Flags: dontcopy
Source: "{#StageDir}\web\*"; DestDir: "{app}\web"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#StageDir}\models\*"; DestDir: "{app}\models"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#StageDir}\runtime\*"; DestDir: "{app}\runtime"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#StageDir}\ffmpeg\*"; DestDir: "{app}\ffmpeg"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#StageDir}\licenses\*"; DestDir: "{app}\licenses"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\AI ADS Studio"; Filename: "{app}\Start.cmd"; WorkingDir: "{app}"; IconFilename: "{app}\ai-ads-studio.ico"; Flags: runminimized
Name: "{autodesktop}\AI ADS Studio"; Filename: "{app}\Start.cmd"; WorkingDir: "{app}"; IconFilename: "{app}\ai-ads-studio.ico"; Flags: runminimized

[Run]
Filename: "{app}\Start.cmd"; WorkingDir: "{app}"; Description: "Start AI ADS Studio"; Flags: postinstall nowait skipifsilent shellexec

[UninstallDelete]
Type: filesandordirs; Name: "{app}\web"
Type: filesandordirs; Name: "{app}\models"
Type: filesandordirs; Name: "{app}\runtime"
Type: filesandordirs; Name: "{app}\ffmpeg"
Type: filesandordirs; Name: "{app}\licenses"
Type: files; Name: "{app}\.env"
Type: files; Name: "{app}\AI-ADS-Studio.exe"
Type: files; Name: "{app}\Start.cmd"
Type: files; Name: "{app}\README.txt"
Type: files; Name: "{app}\ai-ads-studio.ico"

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
var
  GroqKey: AnsiString;
  EnvironmentText: String;
begin
  if CurStep = ssPostInstall then
  begin
    ExtractTemporaryFile('groq-key.txt');
    if not LoadStringFromFile(ExpandConstant('{tmp}\groq-key.txt'), GroqKey) then
      RaiseException('Không thể tải cấu hình tạo kịch bản đi kèm.');
    DeleteFile(ExpandConstant('{tmp}\groq-key.txt'));

    EnvironmentText :=
      'PORT=8080' + #13#10 +
      'LISTEN_ADDR=127.0.0.1:8080' + #13#10 +
      'GROQ_API_KEY=' + GroqKey + #13#10 +
      'GROQ_MODEL={#GroqModel}' + #13#10;
    if not SaveStringToFile(ExpandConstant('{app}\.env'), EnvironmentText, False) then
      RaiseException('Không thể lưu cấu hình phần mềm trên máy này.');
  end;
end;
