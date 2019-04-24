import React, { Component } from 'react';
import Client from './Client.js';
import './App.css';

class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
      ops: [],
    }
  }

  componentDidMount() {
    fetch('http://localhost:8080/?cmd=cat ops.json')
      .then(async r => r.json())
      .then(ops => {
        this.setState({
          ops: Object.entries(ops).map(([k, v]) => {
            return {hash: k, code: v};
          }),
        });
      });
  }

  render() {
    const panelStyle = {display: 'flex', flexDirection: 'column', width: '33%', maxWidth: '33%', margin: '1em'};
    return (
      <div style={{width: '100%', height: '100%', display: 'flex', flexDirection: 'column'}}>
        <h1 style={{margin: '1em 1em 0 1em'}}>Replicant</h1>
        <div style={{display: 'flex', flex: 1, overflow: 'hidden'}}>
          <div style={panelStyle}>
            <Client index={1} ops={this.state.ops}/>
          </div>
          <div style={panelStyle}>
            <Client index={2} ops={this.state.ops}/>
          </div>
          <div style={panelStyle}>
            <h2>Server</h2>
          </div>
        </div>
      </div>
    );
  }
}

export default App;
