import React, { Component } from 'react';

const SYNC_INTERVAL = 3000;

class Server extends Component {
  constructor(props) {
    super(props);
    this.logRef = React.createRef();
    this.state = {
      log: '',
    };
    this.syncTimer = undefined;
  }

  componentDidMount() {
    this.fetch();
  }

  render() {
    return (
      <pre style={{width: '100%', flex: 1, overflow: 'scroll'}} ref={this.logRef}>{this.state.log}</pre>
    );
  }

  async fetch() {
    const url = `http://localhost:8080/exec?cmd=cat server.txt`;
    const r = await fetch(url);
    const t = await r.text();
    if (t != this.state.log) {
      this.setState({
        log: t,
      }, () => {
        this.logRef.current.scrollTop = this.logRef.current.scrollHeight;
      });
    }
    this.syncTimer = window.setTimeout(() => this.fetch(), SYNC_INTERVAL);
  }
}

export default Server;
