import Foundation
import Vision

// Match the desktop's currently committed native OCR configuration.
let imageURL = URL(fileURLWithPath: CommandLine.arguments[1])
let request = VNRecognizeTextRequest()
request.recognitionLevel = .accurate
request.usesLanguageCorrection = false
request.recognitionLanguages = ["zh-Hans", "en-US"]
request.customWords = ["Pmax", "Isc", "Uoc", "W/m2", "W/m²", "mA", "mW", "μA", "μm", "Ω", "℃"]
try VNImageRequestHandler(url: imageURL, options: [:]).perform([request])
let results: [[String: Any]] = (request.results ?? []).compactMap { observation in
    guard let candidate = observation.topCandidates(1).first else { return nil }
    let b = observation.boundingBox
    return ["text": candidate.string, "confidence": candidate.confidence,
            "x": b.origin.x, "y": 1 - b.origin.y - b.height,
            "w": b.width, "h": b.height]
}
let json = try JSONSerialization.data(withJSONObject: results, options: [.prettyPrinted, .sortedKeys])
FileHandle.standardOutput.write(json)
