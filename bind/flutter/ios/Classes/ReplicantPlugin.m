#import "ReplicantPlugin.h"

#import <Repm/Repm.h>

const NSString* CHANNEL_NAME = @"replicant.dev";

NSString* replicantDir(void);

@implementation ReplicantPlugin
  dispatch_queue_t generalQueue;
  dispatch_queue_t syncQueue;

+ (void)registerWithRegistrar:(NSObject<FlutterPluginRegistrar>*)registrar {
  FlutterMethodChannel* channel = [FlutterMethodChannel
      methodChannelWithName:CHANNEL_NAME
            binaryMessenger:[registrar messenger]];
  ReplicantPlugin* instance = [[ReplicantPlugin alloc] init];
  [registrar addMethodCallDelegate:instance channel:channel];

  // Most Replicant operations happen serially, but not blocking UI thread.
  generalQueue = dispatch_queue_create("dev.roci.Replicant", NULL);

  // Sync uses a concurrent queue because we don't want it to block other Replicant operations.
  syncQueue = dispatch_get_global_queue( DISPATCH_QUEUE_PRIORITY_DEFAULT, 0);

  dispatch_async(generalQueue, ^(void){
    NSLog(@"Replicant: init");
    RepmInit(replicantDir(), @"");
  });
}

- (void)handleMethodCall:(FlutterMethodCall*)call result:(FlutterResult)result {
  dispatch_queue_t queue;
  if ([call.method isEqualToString:@"sync"]) {
    queue = syncQueue;
  } else {
    queue = generalQueue;
  }

  NSLog(@"Replicant: Handling: %@ with arguments: %@", call.method, call.arguments);
  dispatch_async(queue, ^(void){
    // The arguments passed from Flutter is a two-element array:
    // 0th element is the name of the database to call on
    // 1st element are the rpc arguments (JSON-encoded)
    NSArray* args = (NSArray*)call.arguments;
    NSError* err = nil;
    NSData* res = RepmDispatch([args objectAtIndex:0], call.method, [[args objectAtIndex:1] dataUsingEncoding:NSUTF8StringEncoding], &err);
    dispatch_async(dispatch_get_main_queue(), ^(void){
      NSLog(@"method: %@", call.method);
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
};

NSString* replicantDir() {
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

  NSError* err;
  [sharedFM createDirectoryAtPath:[dataDir path] withIntermediateDirectories:TRUE attributes:nil error:&err];
  if (err != nil) {
    NSLog(@"Replicant: Could not create data directory: %@", dataDir);
    return nil;
  }

  return [dataDir path];
}

@end
