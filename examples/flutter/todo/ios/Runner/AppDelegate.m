#import <Flutter/Flutter.h>
#import <Repm/Repm.h>
#import "GeneratedPluginRegistrant.h"

#include "AppDelegate.h"
#include "GeneratedPluginRegistrant.h"

@implementation AppDelegate

- (BOOL)application:(UIApplication *)application
    didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
  [GeneratedPluginRegistrant registerWithRegistry:self];
  FlutterViewController* controller = (FlutterViewController*)self.window.rootViewController;
  
  FlutterMethodChannel* channel = [FlutterMethodChannel
                                   methodChannelWithName:@"replicant.dev/examples/todo"
                                   binaryMessenger:controller];

  NSString* dir = [self replicantDir];
  if (dir == nil) {
    return FALSE;
  }
  
  NSLog(@"Replicant: Opening database at: %@", dir);

  NSError *err;
  // TODO: clientid needs to be device-unique
  // Need to generate and store one in user prefs or something.
  RepmConnection *conn = RepmOpen(dir, @"ios/c1", &err);
  if (err != nil) {
    NSLog(@"Replicant: Could not open database: %@", err);
    return FALSE;
  }

  [channel setMethodCallHandler:^(FlutterMethodCall* call, FlutterResult result) {
    NSError *err;
    NSLog(@"Replicant: Sending arguments: %@", call.arguments);
    NSData* res = [conn dispatch:call.method
                            data:[call.arguments dataUsingEncoding:NSUTF8StringEncoding]
                           error: &err];
    if (err != nil) {
      result([FlutterError errorWithCode:@"UNAVAILABLE"
                                 message:@"Error from Replicant"
                                 details:err]);
    } else {
      result([[NSString alloc] initWithData:res
                                   encoding:NSUTF8StringEncoding]);
    }
  }];
  
  [GeneratedPluginRegistrant registerWithRegistry:self];
  return [super application:application didFinishLaunchingWithOptions:launchOptions];
}

- (NSString*)replicantDir {
  NSFileManager* sharedFM = [NSFileManager defaultManager];
  NSArray* possibleURLs = [sharedFM URLsForDirectory:NSApplicationSupportDirectory
                                           inDomains:NSUserDomainMask];
  NSURL* appSupportDir = nil;
  NSURL* dataDir = nil;
  
  if ([possibleURLs count] < 1) {
    NSLog(@"Replicant: Could not location application support directory: %@", dataDir);
    return nil;
  }

  // Use the first directory (if multiple are returned)
  appSupportDir = [possibleURLs objectAtIndex:0];
  NSString* appBundleID = [[NSBundle mainBundle] bundleIdentifier];
  dataDir = [appSupportDir URLByAppendingPathComponent:appBundleID];
  dataDir = [dataDir URLByAppendingPathComponent:@"replicant/db"];
  
  NSError *err;
  [sharedFM createDirectoryAtPath:[dataDir path] withIntermediateDirectories:TRUE attributes:nil error:&err];
  if (err != nil) {
    NSLog(@"Replicant: Could not create data directory: %@", dataDir);
    return nil;
  }
  
  return [dataDir path];
}
@end
