import React, { Component } from 'react';
import {
  Button,
  Image,
  Keyboard,
  Text,
  TextInput,
  TouchableOpacity,
  View
} from "react-native";
import DraggableFlatList from 'react-native-draggable-flatlist'
import Replicant from 'replicant-react-native';
import { styles, viewPadding } from './styles.js'; 
 
export default class App extends Component {
 
  state = {
    text: "",
    todos: [],
  };

  async componentDidMount() {
    this._replicant = new Replicant('https://replicate.to/serve/react-native-susan');
    await this._initBundle();
    this._replicant.onChange = this._load;
    this._load();
  
    Keyboard.addListener(
      "keyboardWillShow",
      e => this.setState({ viewPadding: e.endCoordinates.height + viewPadding })
    );

    Keyboard.addListener(
      "keyboardWillHide",
      () => this.setState({ viewPadding: viewPadding })
    );
  }
 
  renderItem = ({ item, index, move, moveEnd, isActive }) => {
    return (
      <TouchableOpacity onLongPress={move} onPressOut={moveEnd}>
        <View key={item.id}>
          <View style={styles.listItemCont}>
            <Text style={
                [styles.listItem,
                  { textDecorationLine:
                    item.value.done ? 'line-through' : 'none' }]}
              onPress={() => this._handleDone(item.id, item.value.done)}>
              {item.value.title}
            </Text>
          <Button title="X" onPress={() => this._deleteTodo(item.id)} />
          </View>
          <View style={styles.hr} />
        </View>
      </TouchableOpacity>
    )
  }
 
  render() {
    return (
      <View style={[styles.container, { paddingBottom: this.state.viewPadding }]}>
        <TextInput
          ref="addTodoTextInput"
          style={styles.textInput}
          onChangeText={this._handleTextChange}
          onSubmitEditing={this._addTodo}
          value={this.state.text}
          placeholder="Add Tasks"
          returnKeyType="done"
          returnKeyLabel="done"
          autoFocus={true}
          autoCorrect={false}
        />
        <DraggableFlatList
          style={styles.list}
          data={this.state.todos}
          renderItem={this.renderItem}
          keyExtractor={item => item.id}
          scrollPercent={5}
          onMoveEnd={({ to, from }) => this._handleReorder(to, from)}
        />
      </View>
    )
  }

  _initBundle = async () => {
    const resource = require('./replicant.bundle');
    let resolved = Image.resolveAssetSource(resource).uri;

    // EEP. I'm not sure why resolveAssertSource insists on adding an '/assets' dir.
    // I think that it is stripped off internally when this is used with <Image>.
    resolved = resolved.replace('/assets', '');

    const resp = await (await fetch(resolved)).text();
    await this._replicant.putBundle(resp);
  }

  _load = async () => {
    let todos = await this._replicant.exec('getAllTodos');   
    
    // Sort todos by order.
    todos.sort((a, b) => a.value.order - b.value.order);
    
    this.setState({
      todos,
    });
  }

  _handleTextChange = text => {
    this.setState({ text });
  };

  _addTodo = async () => {
    const todos = this.state.todos;
    const text = this.state.text;
    const notEmpty = text.trim().length > 0;

    if (notEmpty) {
      const uid = await this._replicant.exec('uid');
      const index = todos.length == 0 ? 0 : todos.length;
      const order = this._getOrder(index);
      const done = false;
      await this._replicant.exec('addTodo', [uid, text, order, done]);
      this._load();
    }

    // Clear textinput field after todo has been added.
    this.setState({
      text: "",
    });
    
    // Set focus to textInput box after text has been submitted.
    this.refs.addTodoTextInput.focus();
  }

  // Calculates the order field by halving the distance between the left and right neighbor orders.
  // We do this so that order changes still make sense and behave sensibility when clients are making order changes offline.
  _getOrder = (index) => {
    const todos = this.state.todos;
    const minOrderValue = 0;
    const maxOrderValue = Number.MAX_VALUE;
    const leftNeighborOrder = index == 0 ? minOrderValue : todos[index-1].value.order;
    const rightNeighborOrder = index == todos.length ? maxOrderValue : todos[index].value.order;
    const order = leftNeighborOrder + ((rightNeighborOrder - leftNeighborOrder)/2);
    return order;
  }

  _handleDone = async (key, prevDone) => {
    if (key != null) {
      await this._replicant.exec('setDone', [key, !prevDone]);
    }
  };

  _deleteTodo = async (key) => {
    if (key != null) {
      await this._replicant.exec('deleteTodo', [key]);
    }
  };

  _handleReorder = async (to, from) => {
    // There is a bug in the dragdrop library where it can send us to destinations that exceed
    // list length in the case where the items are removed.
    to = Math.min(to, this.state.todos.length - 1);

    const todos = this.state.todos;
    const id = todos[from].id;
    const isMoveup = from > to ? true : false;
    const order = this._getOrder(isMoveup ? to : to + 1);
    await this._replicant.exec('setOrder', [id, order]);
  }
}
