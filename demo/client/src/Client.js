import React, { Component } from 'react';

class Client extends Component {
  constructor(props) {
    super(props);
    this.state = {
      selectedValue: '',
    };
  }
  render() {
    return (
        <div>
            <h2>Client {this.props.index}</h2>
            <select onChange={(e) => this.handleChange_(e)} defaultValue={this.state.selectedValue} style={{width: '100%', marginBottom: '1em'}}>
            {this.props.ops.map((op, i) => {
                return <option key={op.hash} value={op.hash}>{getFunctionName(op.code)}</option>
            })}
                <option key='new' value=''>New...</option>
            </select>
            {this.getFunctionBody()}
            <textarea style={{width: '100%', height: '15em', fontFamily: 'monospace', marginBottom: '1em', background: '#f3f3f3'}} disabled={true}></textarea>
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
      return <textarea style={{width: '100%', height: '15em', fontFamily: 'monospace', whiteSpace: 'pre', marginBottom: '1em'}}/>
    }
    return <pre style={{display:'inline-block', width: '100%', height: '15em', marginBottom: '1em', border: '1px solid grey', overflow:'auto'}}>
      {this.props.ops.find(op => op.hash == this.state.selectedValue).code}
    </pre>
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
