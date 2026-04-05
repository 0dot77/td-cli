# TDCliServer Setup in TouchDesigner

## Quick Setup

1. Open your TouchDesigner project
2. Create a **Base COMP** named `TDCliServer` at the root level (`/project1/TDCliServer`)
3. Inside the COMP, create the following operators:

### Step 1: Web Server DAT
- Create: **Web Server DAT** (`webserver1`)
- Set parameter `port` to `9500`
- Set parameter `active` to `On`

### Step 2: Callbacks Text DAT
- Create: **Text DAT** (`callbacks`)
- Copy the contents of `webserver_callbacks.py` into this DAT
- On the Web Server DAT, set the `Callbacks DAT` parameter to `callbacks`

### Step 3: Handler Text DAT
- Create: **Text DAT** (`handler`)
- Copy the contents of `td_cli_handler.py` into this DAT

### Step 4: Heartbeat Timer
- Create: **Timer CHOP** (`timer1`)
- Set `Length` to `1` seconds
- Set `On Done` to `Re-Start`
- Set `Play` to `On`
- Create: **CHOP Execute DAT** (`chopexec1`)
- Set the CHOP parameter to `timer1`
- Enable `Off to On` callback
- Copy the contents of `heartbeat.py` into the appropriate callbacks
  - `onTimerPulse` goes into the `onOffToOn` callback (fires each 1s cycle)

### Step 5: Verify
- Open a terminal and run: `td-cli status`
- You should see your project info

## Alternative: Textport One-Liner

Paste this into TouchDesigner's Textport (Alt+T):

```python
import urllib.request, os
base = 'https://raw.githubusercontent.com/YOUR_USER/td-cli/main/td/'
dest = project.folder + '/TDCliServer/'
os.makedirs(dest, exist_ok=True)
for f in ['webserver_callbacks.py', 'td_cli_handler.py', 'heartbeat.py']:
    urllib.request.urlretrieve(base + f, dest + f)
print('Files downloaded to', dest)
print('Follow setup_instructions.md to wire up the operators')
```

## Network Layout

```
/project1/TDCliServer/
├── webserver1        (Web Server DAT, port 9500, callbacks -> callbacks)
├── callbacks         (Text DAT, contains webserver_callbacks.py)
├── handler           (Text DAT, contains td_cli_handler.py)
├── timer1            (Timer CHOP, 1s cycle, auto-restart)
└── chopexec1         (CHOP Execute DAT, references timer1, contains heartbeat.py)
```
