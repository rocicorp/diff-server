import React, { Component } from 'react';

class Client extends Component {
  constructor(props) {
    super(props);
    this.state = {
      params: '',
      selectedValue: '',
      dbState: '',
    };
  }

  componentDidMount() {
    this.refreshDBState();
  }

  render() {
    return (
        <div>
            <h2>Client {this.props.index}</h2>
            <select style={{width: '100%', marginBottom: '0.5em'}} 
                onChange={(e) => this.handleChange_(e)}
                defaultValue={this.state.selectedValue}>
              {this.props.ops.map((op, i) => {
                  return <option key={op.hash} value={op.hash}>{getFunctionName(op.code)}</option>
              })}
                <option key='new' value=''>New...</option>
            </select>
            {this.getFunctionBody()}
            <div style={{display: 'flex', alignItems: 'center'}}>
              <div>Params:</div>
              <div style={{flex:1}}><input style={{width:'100%'}} type='text' onChange={(e) => this.handleParamsChange_(e)}/></div>
              <div><button onClick={() => this.handleRun_()}>Run!</button></div>
            </div>
            <pre style={{width: '100%', height: '15em', marginBottom: '1em', background: '#f3f3f3', overflow: 'scroll', border: '1px solid grey'}}>
              {this.state.dbState}
            </pre>
            <div style={{display: 'flex'}}>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <label><input type="checkbox" defaultChecked={true}/>Online</label>
                    <label><input type="checkbox" defaultChecked={true}/>Live</label>
                </div>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <button onClick={() => this.handleSync_()}>Sync</button>
                </div>
            </div>
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
      return <textarea style={{display: 'block', width: '100%', height: '15em', fontFamily: 'monospace', whiteSpace: 'pre', margin: 0}}/>
    }
    return <pre style={{width: '100%', height: '15em', margin: 0, border: '1px solid grey', overflow:'auto', margin: 0}}>
      {this.getFunctionCode()}
    </pre>
  }

  async handleSync_() {
    const url = `http://localhost:8080/exec?cmd=${escape(`replicant sync db${this.props.index} server.txt`)}`;
    console.log(url);
    await fetch(url);
    this.refreshDBState();
  }

  async handleRun_() {
    const url = `http://localhost:8080/exec?cmd=${escape(`replicant op db${this.props.index} ${getFunctionName(this.getFunctionCode())} ${this.state.params}`)}`;
    console.log(url);
    await fetch(url);
    this.refreshDBState();
  }

  async refreshDBState() {
    const url = `http://localhost:8080/exec?cmd=${escape(`noms show db${this.props.index}::local.value`)}`;
    fetch(url).then(r => r.text()).then(t => {
      this.setState({
        dbState: t,
      })
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
