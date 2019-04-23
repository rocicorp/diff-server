import React, { Component } from 'react';

class Client extends Component {
  constructor(props) {
    super(props);
    this.state = {
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
            <select style={{width: '100%', marginBottom: '1em'}} 
                onChange={(e) => this.handleChange_(e)}
                defaultValue={this.state.selectedValue}>
              {this.props.ops.map((op, i) => {
                  return <option key={op.hash} value={op.hash}>{getFunctionName(op.code)}</option>
              })}
                <option key='new' value=''>New...</option>
            </select>
            {this.getFunctionBody()}
            <pre style={{width: '100%', height: '15em', marginBottom: '1em', background: '#f3f3f3', overflow: 'scroll', border: '1px solid grey'}}>
              {this.state.dbState}
            </pre>
            <div style={{display: 'flex'}}>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <label><input type="checkbox" defaultChecked={true}/>Online</label>
                    <label><input type="checkbox" defaultChecked={true}/>Live</label>
                </div>
                <div style={{display: 'flex', flexDirection: 'column', flex: 1}}>
                    <button>Sync</button>
                </div>
            </div>
            <div></div>
        </div>
    );
  }

  handleChange_(e) {
      this.setState({
          selectedValue: e.target.value,
      });
  }

  getFunctionBody() {
    if (!this.state.selectedValue) {
      return <textarea style={{display: 'block', width: '100%', height: '15em', fontFamily: 'monospace', whiteSpace: 'pre', margin: '1em 0'}}/>
    }
    return <pre style={{width: '100%', height: '15em', marginBottom: '1em', border: '1px solid grey', overflow:'auto', margin: '1em 0'}}>
      {this.props.ops.find(op => op.hash == this.state.selectedValue).code}
    </pre>
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
