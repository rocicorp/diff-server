import { StyleSheet } from "react-native";

const viewPadding = 10;

const styles =  StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: "flex-start",
    alignItems: "stretch",
    backgroundColor: "#F5FCFF",
    padding: viewPadding,
    paddingTop: 50,
  },
  listItem: {
    paddingTop: 10,
    paddingBottom: 10,
    fontSize: 14,
    color: "#333333",
  },
  hr: {
    height: 1,
  },
  listItemCont: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between"
  },
  textInput: {
    height: 40,
    paddingRight: 10,
    paddingLeft: 10,
    borderColor: "#AAAAAA",
    borderWidth: 1,
    width: "100%"
  },
});

export { styles, viewPadding };