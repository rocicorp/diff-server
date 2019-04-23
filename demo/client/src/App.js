import React, { Component } from 'react';
import Client from './Client.js';
import './App.css';

class App extends Component {
  constructor() {
    this.state = {
      ops: [],
    }
  }
  render() {
    const panelStyle = {flex: 1, margin: '1em'};
    return (
      <div>
        <h1 style={{margin: '1em 1em 0 1em'}}>Replicant</h1>
        <div style={{display: 'flex'}}>
          <div style={panelStyle}>
            <Client index={1} ops={[]}/>
          </div>
          <div style={panelStyle}>
            <Client index={2} ops={[]}/>
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
