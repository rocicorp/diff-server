import React, { Component } from 'react';

const SYNC_INTERVAL = 3000;

class Client extends Component {
  constructor(props) {
    super(props);
    this.logRef = React.createRef();
    this.state = {
      params: '',
      selectedValue: '',
      dbState: '',
      log: '',
      online: true,
      continuous: false,
    };
    this.syncTimer = undefined;
  }

  static getDerivedStateFromProps(nextProps, prevState) {
    if (!prevState.selectedValue && nextProps.ops.length) {
      prevState.selectedValue = nextProps.ops[0].hash;
    }
  }

  componentDidMount() {
    this.refreshDBState();
  }

  render() {
    return (
        <div style={{display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden'}}>
            <h2>Client {this.props.index}</h2>
            <select style={{width: '100%', marginBottom: '0.5em'}} 
                onChange={(e) => this.handleChange_(e)}
                defaultValue={this.state.selectedValue}>
              {this.props.ops.map((op, i) => {
                  return <option key={op.hash} value={op.hash}>{getFunctionName(op.code)}</option>
              })}
            </select>
            {this.getFunctionBody()}
            <div style={{display: 'flex', alignItems: 'center'}}>
              <div>Params:</div>
              <div style={{flex:1}}><input style={{width:'100%'}} type='text' onChange={(e) => this.handleParamsChange_(e)}/></div>
              <div><button onClick={() => this.handleRun_()}>Run!</button></div>
            </div>
            <pre style={{width: '100%', flex: 1, marginBottom: '0.5em', background: '#f3f3f3', overflow: 'scroll', border: '1px solid grey'}}>
              {this.state.dbState}
            </pre>
            <button onClick={() => this.handleSync_()}>Sync Now</button>
            <div style={{display: 'flex', marginTop: '0.25em'}}>
              <label style={{flex: 1}}><input type="checkbox" defaultChecked={this.state.online}
                  onChange={(e) => this.setState({online: e.target.checked})}/>Online</label>
              <label style={{flex: 1}}><input type="checkbox" defaultChecked={Boolean(this.state.continuous)}
                  onChange={(e) => this.setContinuousSync(e.target.value)}/>Continuous Sync</label>
            </div>
            <pre style={{width: '100%', flex: 1, overflow: 'scroll'}}
              ref={this.logRef}>{this.state.log}</pre>
        </div>
    );
  }

  setContinuousSync(enabled) {
    if (enabled) {
      this.scheduleSync();
    } else {
      this.unscheduleSync();
    }
  }

  handleParamsChange_(e) {
    this.setState({
      params: e.target.value,
    });
  }

  handleChange_(e) {
      this.setState({
          selectedValue: e.target.value,
      });
  }

  getFunctionCode() {
    if (!this.state.selectedValue) {
      return '';
    }
    return this.props.ops.find(op => op.hash === this.state.selectedValue).code;
  }

  getFunctionBody() {
    if (!this.state.selectedValue) {
      return <textarea style={{display: 'block', width: '100%', flex: 1, fontFamily: 'monospace', whiteSpace: 'pre', margin: 0}}/>
    }
    return <pre style={{width: '100%', flex: 1, margin: 0, border: '1px solid grey', overflow:'auto'}}>
      {this.getFunctionCode()}
    </pre>
  }

  async handleSync_() {
    this.unscheduleSync();
    if ((await this.exec(`replicant sync db${this.props.index} server.txt`)) === null) {
      return;
    }
    await this.refreshDBState();
    if (this.state.continuous) {
      this.scheduleSync();
    }
  }

  unscheduleSync() {
    this.syncTimer = window.clearInterval(this.syncTimer);
  }

  scheduleSync() {
    this.syncTimer = window.setTimeout(() => this.handleSync_(), SYNC_INTERVAL);
  }

  async handleRun_() {
    if ((await this.exec(`replicant op db${this.props.index} ${getFunctionName(this.getFunctionCode())} ${this.state.params}`)) === null) {
      return;
    }
    this.refreshDBState();
  }

  async refreshDBState() {
    this.setState({
      dbState: await this.exec(`noms show db${this.props.index}::local.value`),
    });
  }

  async exec(cmd) {
    await this.log('> ' + cmd);
    if (!this.state.online) {
      await this.log('ERROR: client is offline!');
      return null;
    }
    const url = `http://localhost:8080/exec?cmd=${escape(cmd)}`;
    const r = await fetch(url);
    const t = await r.text();
    await this.log(t);
    return t;
  }

  async log(msg) {
    console.log('logging', msg)
    await new Promise((res, rej) => {
      this.setState({
        log: this.state.log + msg + '\n',
      }, () => {
        this.logRef.current.scrollTop = this.logRef.current.scrollHeight;
        res();
      });
    })
  }
}

function getFunctionName(code) {
    const firstLine = code.split('\n')[0];
    const match = firstLine.match(/function(.+?)\(/);
    if (match) {
        const name = match[1].trim();
        if (name) {
            return name;
        }
    }
    return '<anon>';
}

export default Client;
