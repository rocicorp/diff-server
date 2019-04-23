import React, { Component } from 'react';

class Client extends Component {
  render() {
    return (
        <div>
            <h2>Client {this.props.index}</h2>
            <select style={{width: '100%', marginBottom: '1em'}}>
            {this.props.ops.map(op => {
                return <option value={op.hash}>op.name</option>
            })}
                <option>New...</option>
            </select>
            <textarea style={{width: '100%', height: '15em', fontFamily: 'monospace', marginBottom: '1em'}}></textarea> 
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
}

export default Client;
