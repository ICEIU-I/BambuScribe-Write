//go:build darwin

#import <Foundation/Foundation.h>
#import <Vision/Vision.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>

char *bs_ocr_text(const void *bytes, size_t length) {
    @autoreleasepool {
        NSData *data = [NSData dataWithBytes:bytes length:length];
        if (!data) return NULL;

        dispatch_semaphore_t done = dispatch_semaphore_create(0);
        __block NSArray<VNRecognizedTextObservation *> *observations = nil;
        VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc]
            initWithCompletionHandler:^(VNRequest *req, NSError *error) {
                if (!error) observations = [req.results copy];
                dispatch_semaphore_signal(done);
            }];
        request.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
        request.usesLanguageCorrection = NO;
        request.recognitionLanguages = @[@"zh-Hans", @"en-US"];
        request.customWords = @[@"Pmax", @"Isc", @"Uoc", @"W/m2", @"W/m²", @"mA", @"mW", @"μA", @"μm", @"Ω", @"℃"];

        VNImageRequestHandler *handler = [[VNImageRequestHandler alloc]
            initWithData:data options:@{}];
        NSError *error = nil;
        [handler performRequests:@[request] error:&error];
        dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
        if (error || observations.count == 0) return NULL;

        NSMutableArray *result = [NSMutableArray arrayWithCapacity:observations.count];
        for (VNRecognizedTextObservation *observation in observations) {
            VNRecognizedText *candidate = [[observation topCandidates:1] firstObject];
            if (!candidate.string.length) continue;
            CGRect box = observation.boundingBox;
            [result addObject:@{
                @"text": candidate.string,
                @"x": @(box.origin.x),
                @"y": @(1.0 - box.origin.y - box.size.height),
                @"w": @(box.size.width),
                @"h": @(box.size.height),
                @"confidence": @(candidate.confidence)
            }];
        }
        if (!result.count) return NULL;
        NSData *json = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
        if (!json) return NULL;
        return strndup(json.bytes, json.length);
    }
}
