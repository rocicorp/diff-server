# react-native-replicant

## Getting started

`$ npm install react-native-replicant --save`

### Mostly automatic installation

`$ react-native link react-native-replicant`

### Manual installation


#### iOS

1. In XCode, in the project navigator, right click `Libraries` ➜ `Add Files to [your project's name]`
2. Go to `node_modules` ➜ `react-native-replicant` and add `Replicant.xcodeproj`
3. In XCode, in the project navigator, select your project. Add `libReplicant.a` to your project's `Build Phases` ➜ `Link Binary With Libraries`
4. Run your project (`Cmd+R`)<

#### Android

1. Open up `android/app/src/main/java/[...]/MainApplication.java`
  - Add `import dev.roci.ReplicantPackage;` to the imports at the top of the file
  - Add `new ReplicantPackage()` to the list returned by the `getPackages()` method
2. Append the following lines to `android/settings.gradle`:
  	```
  	include ':react-native-replicant'
  	project(':react-native-replicant').projectDir = new File(rootProject.projectDir, 	'../node_modules/react-native-replicant/android')
  	```
3. Insert the following lines inside the dependencies block in `android/app/build.gradle`:
  	```
      compile project(':react-native-replicant')
  	```


## Usage
```javascript
import Replicant from 'react-native-replicant';

// TODO: What to do with the module?
Replicant;
```
