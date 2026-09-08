import {
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListTemplates,
  NewDocumentDraft,
  SaveDocument,
  SaveSettings,
  SaveTemplates,
} from '../wailsjs/go/main/App';
import { model } from '../wailsjs/go/models';
import { showAlert } from './dialog';

export type Document = model.Document;
export type DocumentSummary = model.DocumentSummary;
export type IntrareRow = model.IntrareRow;
export type IesireRow = model.IesireRow;
export type Product = model.Product;
export type Settings = model.Settings;
export type Template = model.Template;

export {
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListTemplates,
  NewDocumentDraft,
  SaveDocument,
  SaveSettings,
  SaveTemplates,
};

/** Shows a Go-side error to the user in Romanian. */
export function showError(prefix: string, err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  void showAlert(`${prefix}: ${message}`);
}
