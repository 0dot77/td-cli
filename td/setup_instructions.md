# TDCliServer Setup in TouchDesigner

## Recommended: Import the `.tox`

The easiest setup is to import [`../tox/TDCliServer.tox`](../tox/TDCliServer.tox) into your TouchDesigner project.

1. Open your TouchDesigner project
2. Import `tox/TDCliServer.tox` into the root network, or drag the file into the project
3. Make sure the imported component is named `TDCliServer`
4. Open the component and verify:
   - `webserver1` is active
   - `webserver1` uses port `9500`
5. Run `td-cli status` in a terminal

If everything is set up correctly, you should see your project info.

## What Is Inside `TDCliServer.tox`

The `.tox` already includes the required network:

```text
/project1/TDCliServer/
├── webserver1        (Web Server DAT, port 9500)
├── callbacks         (Text DAT, web server callbacks)
├── handler           (Text DAT, request handler)
├── timer1            (Timer CHOP, 1 second loop)
└── chopexec1         (CHOP Execute DAT, heartbeat writer)
```

## Manual Setup

If you want to build the component yourself instead of importing the `.tox`, create a **Base COMP** named `TDCliServer` at `/project1/TDCliServer` and add the following:

### Step 1: Web Server DAT

- Create **Web Server DAT** `webserver1`
- Set `port` to `9500`
- Set `active` to `On`

### Step 2: Callbacks Text DAT

- Create **Text DAT** `callbacks`
- Copy the contents of [`webserver_callbacks.py`](webserver_callbacks.py) into this DAT
- On `webserver1`, set `Callbacks DAT` to `callbacks`

### Step 3: Handler Text DAT

- Create **Text DAT** `handler`
- Copy the contents of [`td_cli_handler.py`](td_cli_handler.py) into this DAT

### Step 4: Heartbeat Timer

- Create **Timer CHOP** `timer1`
- Set `Length` to `1` second
- Set `On Done` to `Re-Start`
- Set `Play` to `On`
- Create **CHOP Execute DAT** `chopexec1`
- Set the CHOP parameter to `timer1`
- Enable the `Off to On` callback
- Copy the `onTimerPulse` logic from [`heartbeat.py`](heartbeat.py) into the `onOffToOn` callback

### Step 5: Verify

- Open a terminal and run `td-cli status`
- You should see your project info
