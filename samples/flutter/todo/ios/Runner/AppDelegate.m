#import <Flutter/Flutter.h>
#import <Repm/Repm.h>
#import "GeneratedPluginRegistrant.h"

#include "AppDelegate.h"
#include "GeneratedPluginRegistrant.h"

const NSString* CHANNEL_NAME = @"replicant.dev";

@implementation AppDelegate

  RepmConnection* conn;

- (BOOL)application:(UIApplication *)application
    didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
  [GeneratedPluginRegistrant registerWithRegistry:self];
  FlutterViewController* controller = (FlutterViewController*)self.window.rootViewController;
  
  FlutterMethodChannel* channel = [FlutterMethodChannel
                                   methodChannelWithName:CHANNEL_NAME
                                   binaryMessenger:controller];

  [channel setMethodCallHandler:^(FlutterMethodCall* call, FlutterResult result) {
    dispatch_async(dispatch_get_global_queue( DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^(void){
      [self connect];
      if (conn == nil) {
        result([NSError errorWithDomain:@"Replicant"
                                   code:1
                               userInfo:@{NSLocalizedDescriptionKey: NSLocalizedString(@"Could not open Replicant database.", nil)}]);
        return;
      }

      NSError *err;
      NSLog(@"Replicant: Calling: %@ with arguments: %@", call.method, call.arguments);
      NSData* res = [conn dispatch:call.method
                              data:[call.arguments dataUsingEncoding:NSUTF8StringEncoding]
                             error:&err];
      dispatch_async(dispatch_get_main_queue(), ^(void){
        if (err != nil) {
          result([FlutterError errorWithCode:@"UNAVAILABLE"
                                    message:[err localizedDescription]
                                    details:nil]);
        } else {
          result([[NSString alloc] initWithData:res
                                      encoding:NSUTF8StringEncoding]);
        }
      });
    });
  }];

  return [super application:application didFinishLaunchingWithOptions:launchOptions];
}

- (void)connect {
  @synchronized (self) {
    if (conn != nil) {
      return;
    }

    NSString* repDir = [self replicantDir];
    if (repDir == nil) {
      return;
    }

    NSError *err;
    // TODO: clientid needs to be device-unique
    // Need to generate and store one in user prefs or something.
    conn = RepmOpen(repDir, @"ios/c1", @"", &err);
    if (err != nil) {
      NSLog(@"Replicant: Could not open database: %@", err);
      return;
    }
  }
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
  dataDir = [dataDir URLByAppendingPathComponent:@"replicant"];
  
  NSError *err;
  [sharedFM createDirectoryAtPath:[dataDir path] withIntermediateDirectories:TRUE attributes:nil error:&err];
  if (err != nil) {
    NSLog(@"Replicant: Could not create data directory: %@", dataDir);
    return nil;
  }
  
  return [dataDir path];
}
@end
