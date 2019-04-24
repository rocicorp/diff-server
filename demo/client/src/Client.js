import React, { Component } from 'react';

class Client extends Component {
  constructor(props) {
    super(props);
    this.logRef = React.createRef();
    this.state = {
      params: '',
      selectedValue: '',
      dbState: '',
      log: '',
    };
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
            <pre style={{width: '100%', flex: 1, marginBottom: '1em', background: '#f3f3f3', overflow: 'scroll', border: '1px solid grey'}}>
              {this.state.dbState}
            </pre>
            <div style={{display: 'flex'}}>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <label><input type="checkbox" disabled={true}/>Online</label>
                    <label><input type="checkbox" disabled={true}/>Live</label>
                </div>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <button onClick={() => this.handleSync_()}>Sync</button>
                </div>
            </div>
            <pre style={{width: '100%', flex: 1, overflow: 'scroll', border: '1px solid grey'}}
              ref={this.logRef}>{this.state.log}</pre>
        </div>
    );
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
    return this.props.ops.find(op => op.hash == this.state.selectedValue).code;
  }

  getFunctionBody() {
    if (!this.state.selectedValue) {
      return <textarea style={{display: 'block', width: '100%', flex: 1, fontFamily: 'monospace', whiteSpace: 'pre', margin: 0}}/>
    }
    return <pre style={{width: '100%', flex: 1, margin: 0, border: '1px solid grey', overflow:'auto', margin: 0}}>
      {this.getFunctionCode()}
    </pre>
  }

  async handleSync_() {
    await this.exec(`replicant sync db${this.props.index} server.txt`);
    this.refreshDBState();
  }

  async handleRun_() {
    await this.exec(`replicant op db${this.props.index} ${getFunctionName(this.getFunctionCode())} ${this.state.params}`);
    this.refreshDBState();
  }

  async refreshDBState() {
    this.setState({
      dbState: await this.exec(`noms show db${this.props.index}::local.value`),
    });
  }

  async exec(cmd) {
    this.log('> ' + cmd);
    const url = `http://localhost:8080/exec?cmd=${escape(cmd)}`;
    const r = await fetch(url);
    const t = await r.text();
    this.log(t);
    return t;
  }

  log(msg) {
    this.setState({
      log: this.state.log + msg + '\n',
    }, () => {
      this.logRef.current.scrollTop = this.logRef.current.scrollHeight;
    });
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
